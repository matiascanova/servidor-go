# Aplicación web: Puntuador de materias

Los alumnos de Exactas (por ahora, Ingeniería en Sistemas) puntúan las materias de 1 a 5 en **dificultad**, **relevancia** y **gusto personal**, y dejan una breve opinión. La idea central es que hacer una reseña sea simple, para juntar más datos que con las encuestas actuales. Más adelante habrá login (con modo invitado solo para ver puntajes) y promedios por materia.

> **Estado de este branch:** solo páginas HTML. La API REST en JSON no está resuelta acá (ver sección 3).

---

## 1. Base de datos (PostgreSQL)

Esquema en `sql/schema/schema.sql` y datos de prueba en `sql/schema/seed.sql`. Tres tablas:

- **usuario**: id, nombre y apellido.
- **materia**: id, nombre, cuatrimestre del plan y una descripción corta opcional.
- **voto**: une usuario y materia (clave primaria compuesta y dos claves foráneas con borrado en cascada), con las tres puntuaciones, una descripción de hasta 140 caracteres y la fecha (por defecto, la del día).

Restricciones del `voto`: cada puntuación debe estar entre 1 y 5 o ser NULL, y **al menos una** de las tres debe estar cargada. Así no se guardan votos vacíos, pero la base permite puntuar solo lo que se quiere.

**Datos iniciales:** 2 usuarios (el id 1 es el usuario demo) y 37 materias del plan, en 10 cuatrimestres.

**Consultas:** se escriben en `queries.sql` y sqlc genera el paquete `db`. El voto se guarda con un *upsert*: si ya existe, se reemplaza y se actualiza la fecha. Por eso votar dos veces no da error. Hay consultas de CRUD para usuarios, materias y votos, pero esta versión solo usa las de listar y obtener materias, obtener un voto, listar votos por materia y el upsert.

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
| Hurl | pruebas automáticas de las páginas y del formulario |

### Páginas (HTML)

| Ruta | Qué hace |
|---|---|
| `GET /` | lista de materias |
| `GET /materias/calificar?id=N` | formulario de reseña (precargado si ya votaste) |
| `GET /materias/resenas?id=N` | reseñas de una materia |
| `POST /votos` | guarda el voto y redirige a `/materias/resenas?id=N` (patrón Post/Redirect/Get) |
| `GET /static/...` | archivos estáticos |

Un `id` inválido responde 400 y una materia inexistente, 404.

Las vistas están en `views/`: `layout.templ` (estructura base), `entity_list.templ` (grilla de materias), `entity_form.templ` (formulario) y `materia_resenas.templ` (reseñas).

### Usuario demo y formulario

Como todavía no hay login, los votos se asignan al **usuario demo (id 1)**, que se crea con `seed.sql` al iniciar la base.

El formulario exige las tres puntuaciones (si falta alguna o no es un número, responde 400). La descripción es opcional. Se valida en Go (puntuaciones de 1 a 5, descripción de hasta 140 caracteres) y la base repite las restricciones como última defensa.

### Configuración

- `DATABASE_URL`: conexión a la base. Por defecto, `postgres://postgres:postgres@localhost:5432/apirest?sslmode=disable`.
- Al iniciar, la app reintenta conectarse a la base hasta 15 veces, cada 2 segundos.
- El servidor escucha en el puerto 8080.

---

## 3. API REST (JSON): pendiente

La API en JSON (`/usuarios`, `/materias`, `/votos` con `PUT`, `DELETE` y listados) **no está resuelta en este branch**: `main.go` solo registra las rutas HTML. Las consultas del CRUD ya están generadas en `queries.sql` para cuando se retome.

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
| `VotoDTO` y `validate()` | reciben los datos del formulario y los validan antes de tocar la base |
| `nullInt32`, `nullString` | convierten valores de Go a los tipos NULL de la base |
| `classifyDBError` | traduce errores de Postgres a códigos HTTP con un mensaje legible (duplicado → 409, FK o CHECK → 400, no encontrado → 404, otro → 500) sin mostrar el error crudo al usuario |
| `parseID`, `withQueryID` | leen y validan el `id` de la URL |
| `formInt` | lee un campo numérico del formulario |

---

## 5. Cómo ejecutar y probar

Requisitos: Go 1.22 o superior, Docker con Compose, make, templ, sqlc y Hurl.

```bash
make        # genera el código, levanta la base de datos y corre la app en http://localhost:8080
make test   # corre las pruebas (con la app ya corriendo, en otra terminal)
make down   # detiene la base de datos y borra sus datos
```

`make` solo levanta la base en Docker; la app corre localmente. El servicio `api` de `docker-compose.yml` permite correrla también en un contenedor. Los scripts de `schema.sql` y `seed.sql` solo se ejecutan cuando la base se crea de cero, por eso `make down` borra el volumen.

### Pruebas (`requests.hurl`)

Hurl ejecuta las pruebas en orden y **valida automáticamente** los códigos de estado y el contenido de las respuestas. Cubren:

1. **Páginas HTML:** `GET /`, ruta inexistente (404), formulario vacío y precargado tras votar, y reseñas.
2. **Formulario:** el `POST` válido con su redirección (303) y la reseña publicada.
3. **Casos de error:** IDs inválidos (400), materias inexistentes (404), puntuaciones fuera de rango (400) y valores no numéricos (400).

Las pruebas asumen una base recién creada (la materia 1 sin votos de otros usuarios), así que conviene correr `make down` antes de repetirlas.

---
