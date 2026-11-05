# Aplicación web: Puntuador de materias

Los alumnos de Exactas (por ahora, Ingeniería en Sistemas) puntúan las materias de 1 a 5 en **dificultad**, **relevancia** y **gusto personal**, y dejan una breve opinión. La idea central es que hacer una reseña sea simple, para juntar más datos que con las encuestas actuales. Más adelante habrá login (con modo invitado solo para ver puntajes) y promedios por materia.

---

## 1. Base de datos (PostgreSQL)

Tres tablas:

- **usuario**: id, nombre y apellido.
- **materia**: id, nombre, cuatrimestre del plan y una descripción corta.
- **voto**: une usuario y materia (clave primaria compuesta y dos claves foráneas), con las tres puntuaciones, una descripción de hasta 140 caracteres y la fecha.

Restricciones del `voto`: cada puntuación debe estar entre 1 y 5 o ser NULL, y **al menos una** de las tres debe estar cargada. Así no se guardan votos vacíos, pero se permite puntuar solo lo que se quiere.

**Rendimiento:** la tabla más usada será `voto`. Si el cálculo de promedios en cada consulta se volviera pesado, se podría guardar el promedio en `materia` y actualizarlo con un *stored procedure* (por ejemplo una vez por día), a cambio de algo de consistencia. Por ahora está en evaluación.

**Tests de la base:** `db_test.go` prueba el CRUD de usuarios y votos y un caso mínimo de materias.

---

## 2. Aplicación web

### Tecnologías

| Tecnología | Uso |
|---|---|
| Go (`net/http`) | servidor, rutas y handlers |
| [templ](https://templ.guide) | vistas HTML renderizadas en el servidor |
| PostgreSQL + `pgx` | base de datos |
| sqlc | genera el código Go de acceso a datos (paquete `db`) desde `queries.sql` |
| Docker Compose | PostgreSQL (y opcionalmente la app) en contenedores |
| Hurl | pruebas automáticas de la API y de las páginas |

### Páginas (HTML)

| Ruta | Qué hace |
|---|---|
| `GET /` | lista de materias |
| `GET /materias/calificar?id=N` | formulario de reseña (precargado si ya votaste) |
| `GET /materias/resenas?id=N` | reseñas de una materia |
| `POST /votos` | guarda el voto y redirige a `/` (patrón Post/Redirect/Get) |

Las vistas están en `views/`: `layout.templ` (estructura base), `entity_list.templ` (grilla de materias), `entity_form.templ` (formulario) y `materia_resenas.templ` (reseñas).

Como todavía no hay login, los votos del formulario se asignan al **usuario demo (id 1)**, que se crea al iniciar la app si no existe.

---

## 3. API REST (JSON)

Además de las páginas, la app expone un CRUD en JSON. Los errores devuelven `{"error": "mensaje"}`.

**Usuarios** (`/usuarios`) y **Materias** (`/materias`)

| Método | Ruta | Respuestas |
|---|---|---|
| GET | `/usuarios` | 200 |
| POST | `/usuarios` | 201, 400, 409 |
| GET | `/usuarios/{id}` | 200, 400, 404 |
| PUT | `/usuarios/{id}` | 200, 400, 404 |
| DELETE | `/usuarios/{id}` | 204 |

Las materias tienen las mismas rutas y códigos bajo `/materias`.

**Votos**

| Método | Ruta | Respuestas |
|---|---|---|
| POST | `/votos` | 201, 400, 409 |
| GET | `/votos?idmateria=N` | 200, 400 |
| GET | `/votos/{idusuario}/{idmateria}` | 200, 400, 404 |
| PUT | `/votos/{idusuario}/{idmateria}` | 200, 400, 404 |
| DELETE | `/votos/{idusuario}/{idmateria}` | 204 |

- `POST /votos` atiende el formulario HTML o JSON, según el `Content-Type`.
- No existe un listado de *todos* los votos: solo por materia.
- Puntuaciones fuera de 1 a 5, o un voto sin ninguna puntuación, se rechazan con 400.
- Un `409` indica un recurso duplicado (por ejemplo, votar dos veces con `POST`; para modificar se usa `PUT`).
- `DELETE` es idempotente: responde 204 exista o no el recurso.

---

## 4. Funciones auxiliares (helpers)

Los helpers evitan repetir código y esconden dos incomodidades: los valores NULL de la base y los errores de Postgres.

### Vistas (`views/helpers.go`)

**Por qué existen:** con `database/sql`, sqlc representa las columnas que admiten NULL como `sql.NullInt32` y `sql.NullString`, que son structs con un campo `Valid`. Si el template mostrara directamente `.Int32`, un NULL aparecería como `0/5`, y habría que escribir un `if/else` por cada campo del formulario.

| Helper | Qué hace |
|---|---|
| `scoreDif`, `scoreRel`, `scoreGusto` | valor inicial de cada input (vacío si no hay voto o es NULL) |
| `comentarioVoto` | valor inicial del textarea |
| `formTitle` | título de la página: "Editar" o "Calificar" |
| `scoreLabel` | muestra `4/5`, o `—` si la puntuación es NULL |

También hay un subcomponente `scoreInput` en `entity_form.templ`, que evita repetir tres veces el mismo bloque de HTML.

### Servidor (`main.go`)

| Helper | Por qué |
|---|---|
| DTOs (`MateriaDTO`, `VotoDTO`) y conversiones (`toVotoDTO`, `intPtr`...) | el JSON sale limpio (`"puntuacion_rel": null`) y no como `{"Int32": 3, "Valid": true}` |
| `writeJSON`, `writeError`, `decodeJSON` | respuestas y errores con el mismo formato, sin repetir código en cada handler |
| `classifyDBError` | traduce errores de Postgres a códigos HTTP (duplicado → 409, FK o CHECK → 400) sin mostrar el error crudo al cliente |
| `pathID`, `queryID` | leen y validan los IDs de la ruta o de la URL |

---

## 5. Cómo ejecutar y probar

Requisitos: Go 1.22 o superior, Docker con Compose, make, templ, sqlc y Hurl.

```bash
make        # genera el código, levanta la base de datos y corre la app en http://localhost:8080
make test   # corre las pruebas (con la app ya corriendo, en otra terminal)
make down   # detiene la base de datos y borra sus datos
```

### Pruebas (`requests.hurl`)

Hurl ejecuta las pruebas en orden y **valida automáticamente** los códigos de estado y el contenido de las respuestas. Cubren:

1. **Usuarios y materias:** CRUD completo, verificando con un `GET` que los `PUT` realmente se guardaron.
2. **Páginas HTML:** `GET /`, formulario vacío y precargado, reseñas, y el `POST` del formulario con su redirección.
3. **Votos (JSON):** CRUD, listado por materia y valores `null`.
4. **Casos de error:** IDs inválidos, JSON inválido, recursos inexistentes (404), duplicados (409) y puntuaciones fuera de rango (400).
