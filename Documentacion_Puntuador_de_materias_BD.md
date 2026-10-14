# Aplicación web: Puntuador de materias

La principal idea de esta página es que los alumnos de exactas puedan puntuar las materias (1-5 estrellas) en aspectos de dificultad, relevancia y actualización de contenido y gusto personal, la carrera "ingeniería en sistemas" en principio. Se va a poder ver el puntaje promedio de cada materia y cada alumno va a poder dar una breve opinión sobre la materia y de la materia en sí. Este sistema va a contar con un sistema de login para evitar que ningún alumno escriba reseñas pero se va a contar con una opción de entrar como invitado para revisar los puntajes pero no para puntuar. La fortaleza principal del sistema es que debe ser simple de hacer una reseña para conseguir un volumen mayor de datos confiables al que se consigue con las encuestas actuales.

---

## 1. Base de datos

La Base de datos consiste en un esquema de 3 tablas en donde se guarda la información sobre los usuarios con campos sobre el nombre, apellido, e id, la tabla materia que contiene id, nombre, cuatrimestre (int que indica en qué cuatrimestre está ubicada en el plan de la carrera) y una corta descripción. Y la tabla voto consiste en conectar las otras tablas de forma que se registren las puntuaciones en las distintas materias de los distintos usuarios guardando id materia e id usuario como foreign keys, las 3 categorías en las que se puntúa, una fecha y una corta descripción. Además hay una restricción que no permite al usuario cargar una votación dejando los 3 campos de puntuación libres para no cargar tuplas sin la información que nos interesa pero permitiendo dar esa libertad para cargar solo campos elegidos.

- **El diseño de la DB se encuentra subida al repositorio**

### Consideraciones de rendimiento

La tabla más consultada y modificada será la tabla voto dado que ahí se cargaran los votos de los usuarios. Se consultaran los votos y promedios en relación a las materias. Se está evaluando si es necesario cargar la tabla materia con los promedios generados a partir de los votos dado que en un momento de mucho tráfico puede resultar muy laborioso para el sistema calcular constantemente los promedios solicitados en cada consulta. Esta implementación permitirá que las consultas sobre los promedios sean mucho más eficientes sacrificando la eficiencia en la carga dado que será el momento donde se recalculen los promedios, pero considerando que será menos habitual este caso contando con que solo usuarios registrados deberían poder cargar votos creemos que es la mejor opción. Aun así Postgre permite generar stored procedures que permiten actualizar los promedios una cantidad controlable de veces por lo que, aunque se sacrifique un poco de consistencia podría hacerse una solución que actualice los promedios una vez por dia por ejemplo (si los hay) y nos evitaremos recalcular los promedios cada vez que alguien actualiza.

---

## 2. Test de la base de datos

El código de go de testeo db test.go realiza el testeo principal en la tabla de usuarios, crea dos usuarios y los inserta y trae de la tabla, verificando si los nombres guardados son correctos e informando en caso de que no lo sean. En la siguiente etapa actualiza algunos aspectos de los usuarios y verifica si los nombres fueron actualizados correctamente. Y como último llama a la función de Delete para borrar los dos usuarios creados. Por otro lado hay testeos sobre la tabla votos los típicos CRUD y también se hizo un testeo más simple sobre la tabla materia para asegurar el funcionamiento mínimo.

---

## 3. API REST

Sobre la base de datos se implementó una API REST en Go (`main.go`) que expone operaciones CRUD para las tres entidades.

### Tecnologías

- **Go**: lenguaje del servidor, usando la librería estándar `net/http` (`http.ServeMux`).
- **PostgreSQL**: base de datos, accedida con el driver `pgx` (`database/sql`).
- **sqlc**: genera el código Go de acceso a datos (paquete `db`) a partir de las consultas SQL.
- **Docker y Docker Compose**: levantan la API y la base de datos en contenedores.


### Endpoints

**Usuarios**

| Método | Ruta | Descripción | Respuestas |
|---|---|---|---|
| GET | `/usuarios` | Lista todos los usuarios | 200 |
| POST | `/usuarios` | Crea un usuario | 201, 400 |
| GET | `/usuarios/{id}` | Obtiene un usuario | 200, 404 |
| PUT | `/usuarios/{id}` | Actualiza un usuario | 200, 400, 404 |
| DELETE | `/usuarios/{id}` | Elimina un usuario | 204 |


**Materias**

| Método | Ruta | Descripción | Respuestas |
|---|---|---|---|
| GET | `/materias` | Lista todas las materias | 200 |
| POST | `/materias` | Crea una materia | 201, 400 |
| GET | `/materias/{id}` | Obtiene una materia | 200, 404 |
| PUT | `/materias/{id}` | Actualiza una materia | 200, 400, 404 |
| DELETE | `/materias/{id}` | Elimina una materia | 204 |


**Votos**

| Método | Ruta | Descripción | Respuestas |
|---|---|---|---|
| POST | `/votos` | Crea un voto | 201, 400 |
| GET | `/votos/{idusuario}/{idmateria}` | Obtiene un voto | 200, 404 |
| PUT | `/votos/{idusuario}/{idmateria}` | Actualiza un voto | 200, 404 |
| DELETE | `/votos/{idusuario}/{idmateria}` | Elimina un voto | 204 |


- No existe un `GET /votos` que liste todos los votos.

---

## 4. Pruebas de la API (`requests.sh`)

El script `requests.sh` automatiza la prueba completa de la API.

### Requisitos previos

Se asume que el usuario ya tiene instalado:

- Docker
- Docker Compose
- Go
- sqlc

### Pasos

1. **Iniciar Docker** antes de ejecutar `requests.sh`.

2. Una vez clonado el repositorio, darle permiso de ejecución al archivo `requests.sh`:

   ```bash
   chmod +x requests.sh
   ```

3. Ejecutar el script de testeo desde la consola:

   ```bash
   ./requests.sh
   ```

### Qué hace el script

1. Elimina ejecuciones previas y volúmenes (`docker compose down -v`).
2. Levanta la infraestructura (`docker compose up -d --build`).
3. Espera a que PostgreSQL (contenedor `postgres-db`, base `apirest`) esté listo con `pg_isready`.
4. Espera hasta 20 segundos a que la API responda en `http://localhost:8080`; si no responde, muestra los logs y termina con error.
5. Ejecuta 15 requests HTTP con `curl`, en este orden:

| # | Request | Recurso |
|---|---|---|
| 1 | POST /usuarios | Usuarios |
| 2 | GET /usuarios | Usuarios |
| 3 | GET /usuarios/10 | Usuarios |
| 4 | PUT /usuarios/10 | Usuarios |
| 5 | POST /materias | Materias |
| 6 | GET /materias | Materias |
| 7 | GET /materias/101 | Materias |
| 8 | PUT /materias/101 | Materias |
| 9 | POST /votos | Votos |
| 10 | GET /votos/10/101 | Votos |
| 11 | PUT /votos/10/101 | Votos |
| 12 | DELETE /votos/10/101 | Votos |
| 13 | GET /votos/10/101 (se espera 404) | Votos |
| 14 | DELETE /materias/101 | Materias |
| 15 | DELETE /usuarios/10 | Usuarios |

6. Al finalizar (con éxito o con error) limpia los contenedores y volúmenes de Docker.

El script no valida automáticamente los códigos de respuesta: muestra los encabezados y cuerpos (`curl -i`) para que se revisen manualmente.

---

## 5. Estado actual respecto del diseño original

| Funcionalidad planteada | Estado |
|---|---|
| CRUD de usuarios, materias y votos | Implementado |
| Puntuación de 1 a 5 en dificultad, relevancia y gusto | Implementado (valores fuera de rango se guardan como NULL) |
| Promedio de puntajes por materia | Pendiente (no hay endpoint ni campo de promedio) |
| Login y modo invitado | Pendiente (la API no tiene autenticación) |
| Actualización periódica de promedios con stored procedures | En evaluación |
