# Documentación – Foro de Alumnos (Exactas UNICEN)

## 1. Descripción general

La aplicación es un foro para que los alumnos de Ingeniería en Sistemas califiquen las materias del plan de estudios. Cada materia se puntúa en tres dimensiones, de 1 a 5:

- **Dificultad**
- **Relevancia**
- **Gusto personal**

Además, cada voto puede incluir un comentario de hasta 140 caracteres. Un usuario tiene **un único voto por materia**; si vuelve a votar, su voto se actualiza.

La aplicación ofrece dos interfaces sobre el mismo servidor:

- **Vistas web HTML** para los alumnos: plan de materias, formulario de calificación y reseñas.
- **API REST JSON** para gestionar usuarios, materias y votos.

---

## 2. Tecnologías

| Componente | Uso |
|---|---|
| [Go](https://go.dev) | Servidor HTTP y lógica (`net/http`, sin framework) |
| [PostgreSQL 17](https://www.postgresql.org) | Base de datos relacional |
| [pgx](https://github.com/jackc/pgx) v5 | Driver de PostgreSQL, usado a través de `database/sql` |
| [sqlc](https://sqlc.dev) | Genera el código Go de acceso a datos a partir de SQL |
| [templ](https://templ.guide) v0.2.778 | Plantillas HTML tipadas para las vistas |
| [Docker Compose](https://docs.docker.com/compose/) | Orquestación de la API y la base de datos |
| [Hurl](https://hurl.dev) | Pruebas de integración HTTP |

---

## 3. Arquitectura y estructura del proyecto

### Flujo de una petición

```
Navegador / cliente HTTP
        │
        ▼
 net/http (ServeMux)  ──►  Handlers en main.go
                              │            │
                              │            └──►  views (templ)  ──►  HTML
                              ▼
                       db.Queries (sqlc)
                              │
                              ▼
                        PostgreSQL
```

Un único binario (`main.go`) contiene el servidor, el enrutador y los handlers. Los handlers usan las funciones generadas por sqlc para consultar la base y, según la ruta, responden JSON o renderizan una vista templ.



Los archivos de `db/` y los `*_templ.go` de `views/` son **generados**. Para cambiarlos hay que editar el `.sql` o el `.templ` correspondiente y regenerar (ver [Generación de código](#10-generación-de-código)).

### Estructuras principales de `main.go`

| Estructura | Función |
|---|---|
| `Server` | Agrupa las dependencias: las consultas sqlc (`queries`) y la conexión (`dbConn`) |
| `UsuarioDTO` | Cuerpo JSON de entrada de usuarios: `idusuario`, `nombre`, `apellido` |
| `MateriaDTO` | Cuerpo JSON de entrada de materias: `idmateria`, `nombre`, `cuatrimestre`, `descripcion` |
| `VotoDTO` | Cuerpo JSON de entrada de votos: `idusuario`, `idmateria`, `puntuacion_dif`, `puntuacion_rel`, `puntuacion_gusto`, `descripcion` |

### Conexión a la base

`initDB()` abre un pool de conexiones con estos límites: 25 conexiones abiertas como máximo, 5 inactivas y 5 minutos de vida por conexión. Verifica la conexión con un `Ping()` al arrancar; si falla, el servidor termina con error.

El servidor HTTP usa timeouts de lectura y escritura de 10 segundos y de inactividad de 60 segundos.

---

## 4. Servicios Docker

`docker-compose.yml` define dos servicios:

| Servicio | Contenedor | Imagen | Puerto | Descripción |
|---|---|---|---|---|
| `database` | `postgres-db` | `postgres:17-alpine` | `5432` | Base `apirest`. Al crearse por primera vez ejecuta `schema.sql` y luego `seed.sql`. Los datos persisten en el volumen `pgdata`. |
| `api` | `go-api` | Construida con el `Dockerfile` | `8080` | Servidor Go. Arranca cuando la base supera su *healthcheck*. |

**Healthcheck de la base:** ejecuta `pg_isready -U postgres -d apirest` cada 5 segundos, con 10 reintentos y un período inicial de gracia de 30 segundos. El servicio `api` declara `depends_on` con `condition: service_healthy`.

**Scripts de inicialización:** los archivos montados en `docker-entrypoint-initdb.d` solo se ejecutan cuando el volumen de datos está vacío. Si se modifican `schema.sql` o `seed.sql`, hay que eliminar el volumen (`make down`) para que se apliquen.

**Imagen de la API (multi-etapa):**

1. *Etapa de compilación* (`golang:1.26-alpine`): instala `templ`, ejecuta `templ generate` y compila el binario estático con `CGO_ENABLED=0`.
2. *Etapa final* (`alpine:3.20`): copia solo el ejecutable y la carpeta `static/`.

Como `static/` queda dentro de la imagen, un cambio en el CSS requiere reconstruir la imagen. El `Dockerfile` **no ejecuta `sqlc generate`**, por lo que `db/` debe estar generado y actualizado en el repositorio antes de construir.

---

## 5. Modelo de datos

```
usuario (idusuario PK, nombre, apellido)

materia (idmateria PK, nombre, cuatrimestre, descripcion)

voto    (idusuario PK/FK → usuario,
         idmateria PK/FK → materia,
         puntuacion_dif, puntuacion_rel, puntuacion_gusto,
         descripcion, fecha_voto)
```

### Tablas

**`usuario`**

| Columna | Tipo | Restricciones |
|---|---|---|
| `idusuario` | int | Clave primaria |
| `nombre` | varchar(50) | Obligatorio |
| `apellido` | varchar(50) | Obligatorio |

**`materia`**

| Columna | Tipo | Restricciones |
|---|---|---|
| `idmateria` | int | Clave primaria |
| `nombre` | varchar(50) | Obligatorio |
| `cuatrimestre` | int | Obligatorio |
| `descripcion` | varchar(50) | Opcional |

**`voto`**

| Columna | Tipo | Restricciones |
|---|---|---|
| `idusuario` | int | Clave primaria compuesta y clave foránea a `usuario` |
| `idmateria` | int | Clave primaria compuesta y clave foránea a `materia` |
| `puntuacion_dif` | int | Opcional, entre 1 y 5 |
| `puntuacion_rel` | int | Opcional, entre 1 y 5 |
| `puntuacion_gusto` | int | Opcional, entre 1 y 5 |
| `descripcion` | varchar(140) | Opcional |
| `fecha_voto` | date | Obligatorio, por defecto la fecha actual |

### Reglas de integridad

| Restricción | Efecto |
|---|---|
| `voto_pk (idusuario, idmateria)` | Un voto por usuario y materia |
| `chk_rango_votos` | Cada puntuación debe estar entre 1 y 5, o ser nula |
| `chk_al_menos_un_voto` | Al menos una de las tres puntuaciones debe tener valor |
| `fk_usuario` | El usuario del voto debe existir. `ON UPDATE CASCADE` y `ON DELETE CASCADE` |
| `fk_materia` | La materia del voto debe existir. `ON UPDATE CASCADE` y `ON DELETE CASCADE` |

Al eliminar un usuario o una materia se eliminan también todos sus votos.

### Datos iniciales (`seed.sql`)

El script de datos iniciales carga dos usuarios base y las materias de la carrera. Para las pruebas y esta primera instancia que no cuenta con inicio de sesion se utiliza la id: 1 

---

## 6. API REST

URL base: `http://localhost:8080`. Los cuerpos de request y response son JSON.

### Usuarios

| Método | Ruta | Descripción | Éxito |
|---|---|---|---|
| `GET` | `/usuarios` | Lista todos los usuarios | `200` |
| `POST` | `/usuarios` | Crea un usuario | `201` |
| `GET` | `/usuarios/{id}` | Obtiene un usuario | `200` |
| `PUT` | `/usuarios/{id}` | Actualiza nombre y apellido | `200` |
| `DELETE` | `/usuarios/{id}` | Elimina un usuario | `204` |

Cuerpo de `POST` y `PUT`:

```json
{
  "idusuario": 10,
  "nombre": "Carlos",
  "apellido": "Gómez"
}
```

En `PUT`, el identificador se toma de la URL. Los campos que no forman parte del DTO (por ejemplo, `curso`) se ignoran.

### Materias

| Método | Ruta | Descripción | Éxito |
|---|---|---|---|
| `GET` | `/materias` | Lista todas las materias | `200` |
| `POST` | `/materias` | Crea una materia | `201` |
| `GET` | `/materias/{id}` | Obtiene una materia | `200` |
| `PUT` | `/materias/{id}` | Actualiza una materia | `200` |
| `DELETE` | `/materias/{id}` | Elimina una materia | `204` |

Cuerpo de `POST` y `PUT`:

```json
{
  "idmateria": 101,
  "nombre": "Programación Web",
  "cuatrimestre": 1,
  "descripcion": "La materia del temido zunino"
}
```

### Votos

| Método | Ruta | Descripción | Éxito |
|---|---|---|---|
| `POST` | `/votos` | Crea o actualiza un voto (JSON) | `201` |
| `GET` | `/votos/{idusuario}/{idmateria}` | Obtiene un voto | `200` |
| `PUT` | `/votos/{idusuario}/{idmateria}` | Actualiza un voto | `200` |
| `DELETE` | `/votos/{idusuario}/{idmateria}` | Elimina un voto | `204` |

Cuerpo de `POST` y `PUT`:

```json
{
  "idusuario": 10,
  "idmateria": 101,
  "puntuacion_dif": 3,
  "puntuacion_rel": 4,
  "puntuacion_gusto": 5,
  "descripcion": "Uso linux ahora"
}
```

El `POST /votos` realiza una inserción o actualización (*upsert*): si ya existe un voto para ese usuario y materia, lo reemplaza. En las rutas con identificador, el usuario y la materia se toman de la URL.

**Ruta compartida entre formulario y API.** `POST /votos` atiende tanto el formulario HTML como la API JSON. El servidor los distingue por el header `Content-Type`: se trata como API solo si vale **exactamente** `application/json`. Cualquier otro valor se interpreta como el envío del formulario web.

### Formato de las respuestas

Las respuestas `GET` devuelven las estructuras generadas por sqlc (con etiquetas JSON, `emit_json_tags: true`). Las columnas opcionales (puntuaciones y descripciones) se serializan como objetos con el valor y un indicador de validez:

```json
"descripcion": { "String": "Excelente materia", "Valid": true }
```

El `POST /votos` responde `{"mensaje": "votación exitosa"}`.

---

## 7. Vistas web

| Ruta | Descripción |
|---|---|
| `GET /` | Página principal con la grilla de materias del plan |
| `GET /materias/calificar?id={idmateria}` | Formulario de calificación. Si el usuario ya votó, carga su voto |
| `POST /votos` (formulario) | Guarda el voto y redirige (`303`) a las reseñas de la materia |
| `GET /materias/resenas?id={idmateria}` | Lista las reseñas y puntuaciones de la materia |
| `GET /static/...` | Archivos estáticos (CSS) |

Las vistas se definen en archivos `.templ` dentro de `views/` y se compilan a Go. Los estilos están en `static/styles.css`; las tarjetas de materia, la cabecera y los botones comparten la paleta de verdes azulados definida con variables CSS en `:root`.

El formulario registra los votos siempre con el **usuario de id 1**; no hay autenticación ni sesiones.

---

## 8. Validación y manejo de errores

La integridad de los datos se garantiza en capas:

| Capa | Mecanismo |
|---|---|
| Navegador | El formulario web usa atributos HTML (`min`, `max`) en los campos numéricos, de modo que no se puede enviar una puntuación fuera de 1–5 |
| Handlers de Go | Validan el JSON, y que los identificadores de la URL sean números enteros positivos |
| Base de datos | Restricciones `CHECK`, claves primarias y foráneas, longitudes y `NOT NULL` (ver [Reglas de integridad](#reglas-de-integridad)) |

La base de datos es la garantía final: cualquier dato que se envíe salteando las capas anteriores (por ejemplo, con `curl`) es rechazado por sus restricciones.

### Códigos de respuesta de error

| Código | Cuándo ocurre |
|---|---|
| `400 Bad Request` | JSON inválido o identificador que no es un número positivo |
| `404 Not Found` | El recurso no existe |
| `405 Method Not Allowed` | Método HTTP no soportado en esa ruta |
| `500 Internal Server Error` | Fallo de base de datos, incluidas las violaciones de restricciones (puntuación fuera de rango, usuario o materia inexistente, clave duplicada) |

Las respuestas de error son texto plano con una descripción breve (por ejemplo, `Voto no encontrado`).

---

## 9. Configuración

La aplicación se configura con variables de entorno:

| Variable | Valor por defecto | Descripción |
|---|---|---|
| `DATABASE_URL` | `postgres://postgres:postgres@localhost:5432/apirest?sslmode=disable` | Cadena de conexión a PostgreSQL |
| `PORT` | `8080` | Puerto del servidor HTTP |

En Docker Compose, la API recibe `DATABASE_URL=postgres://postgres:postgres@database:5432/apirest?sslmode=disable`, que apunta al servicio `database` por su nombre.

Credenciales de la base de desarrollo: usuario `postgres`, contraseña `postgres`, base `apirest`. Son valores para desarrollo y no deben usarse en producción.

---

## 10. Pruebas automatizadas

El archivo `requests.hurl` contiene la suite de integración. Las entradas se ejecutan en orden y comparten estado en la base de datos.

| Sección | Cobertura |
|---|---|
| Usuarios | Crear, listar, obtener y actualizar; verifica estado HTTP y contenido |
| Materias | Crear, listar, obtener y actualizar |
| Votos | Crear, obtener por clave compuesta, actualizar, eliminar y verificar el `404` posterior |
| Limpieza | Elimina la materia y el usuario creados por la propia prueba |
| Casos de error | Entradas con JSON inválido y otros casos negativos |
