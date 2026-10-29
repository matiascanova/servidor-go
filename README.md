# Foro de Alumnos – Exactas UNICEN

Aplicación web y API REST para calificar y reseñar las materias de Ingeniería en Sistemas.

Esta guía explica **cómo ejecutar la aplicación y sus funciones**. Para conocer cómo está construida (arquitectura, base de datos, endpoints, reglas de validación), ver [`DOCUMENTACION.md`](DOCUMENTACION.md).

## Contenido

1. [Requisitos previos](#1-requisitos-previos)
2. [Ejecutar la aplicación](#2-ejecutar-la-aplicación)
3. [Usar la aplicación](#3-usar-la-aplicación)
4. [Ejecutar las pruebas](#4-ejecutar-las-pruebas)
5. [Comandos del Makefile](#5-comandos-del-makefile)
6. [Ejecución local de la API (sin contenedor)](#6-ejecución-local-de-la-api-sin-contenedor)
7. [Solución de problemas](#7-solución-de-problemas)

---

## 1. Requisitos previos

### Obligatorios

| Herramienta | Para qué | Verificar con |
|---|---|---|
| **Docker** (Docker Desktop en Windows/macOS, Docker Engine en Linux) con **Docker Compose v2** | Levantar la base de datos y la API | `docker --version` y `docker compose version` |
| **Make** | Usar los atajos del `Makefile` | `make --version` |
| **curl** | Lo usa `make wait` para esperar a la API | `curl --version` |

> ⚠️ **Docker tiene que estar iniciado antes de ejecutar cualquier comando.**
> En Windows y macOS, abrí Docker Desktop y esperá a que indique que está en ejecución. En Linux, iniciá el servicio con `sudo systemctl start docker`.
> Para comprobarlo ejecutá `docker info`: si muestra *"Cannot connect to the Docker daemon"*, Docker no está corriendo.

### Para ejecutar las pruebas

| Herramienta | Verificar con | Instalación |
|---|---|---|
| **Hurl** | `hurl --version` | `brew install hurl` (macOS), `sudo apt install hurl` (Debian/Ubuntu) o <https://hurl.dev> |

### Opcionales (solo para modificar el código o ejecutar la API fuera de Docker)

| Herramienta | Para qué |
|---|---|
| **Go** | Ejecutar la API con `go run` y correr `go test` |
| **sqlc** | Regenerar el paquete `db/` al cambiar `sql/schema` o `sql/queries` |
| **templ** v0.2.778 | Regenerar las vistas al cambiar archivos `.templ` |

> **Windows:** `make` no viene instalado. Lo más simple es usar **WSL 2** (Ubuntu) con Docker Desktop integrado, o **Git Bash**, e instalar `make` ahí.

---

## 2. Ejecutar la aplicación

1. **Iniciá Docker** y confirmá con `docker info`.

2. **Ubicate en la carpeta del proyecto** (la que contiene el `Makefile` y el `docker-compose.yml`):

   ```bash
   cd ruta/al/proyecto
   ```

3. **Levantá la base de datos y la API:**

   ```bash
   make up
   ```

   La primera vez tarda más porque descarga imágenes y compila la aplicación.

4. **Esperá a que la API responda:**

   ```bash
   make wait
   ```

   Cuando termina muestra `API en línea.`

5. **Abrí el navegador** en <http://localhost:8080>.

6. **Ver los logs** (opcional):

   ```bash
   make logs
   ```

7. **Detener todo:**

   ```bash
   make down
   ```

> ⚠️ `make down` **borra los datos de la base**. La próxima vez que levantes el proyecto, la base se crea de nuevo desde cero.

Para reiniciar de cero en un solo paso: `make restart`.

---

## 3. Usar la aplicación

### Desde el navegador

| Qué querés hacer | Cómo |
|---|---|
| Ver el plan de materias | Abrí <http://localhost:8080> |
| Calificar una materia | En la tarjeta de la materia, usá el botón de calificar. Completá dificultad, relevancia y gusto (valores de **1 a 5**) y, opcionalmente, un comentario. Presioná **Guardar Cambios** |
| Editar tu calificación | Volvé a entrar a calificar la misma materia: el formulario aparece con tu voto cargado |
| Ver las reseñas de una materia | En la tarjeta de la materia, usá el botón de reseñas, o abrí `http://localhost:8080/materias/resenas?id=<idmateria>` |

### Desde la API (línea de comandos)

Con la aplicación levantada:

```bash
# Crear un usuario
curl -X POST http://localhost:8080/usuarios \
  -H "Content-Type: application/json" \
  -d '{"idusuario": 10, "nombre": "Carlos", "apellido": "Gómez"}'

# Consultarlo
curl http://localhost:8080/usuarios/10

# Listar materias
curl http://localhost:8080/materias

# Votar una materia
curl -X POST http://localhost:8080/votos \
  -H "Content-Type: application/json" \
  -d '{"idusuario": 10, "idmateria": 101, "puntuacion_dif": 3, "puntuacion_rel": 4, "puntuacion_gusto": 5, "descripcion": "Excelente materia"}'

# Ver ese voto
curl http://localhost:8080/votos/10/101
```

> Para votar, el usuario y la materia **tienen que existir** y las puntuaciones deben estar entre 1 y 5. El header `Content-Type: application/json` es obligatorio en `POST /votos`.

La lista completa de endpoints está en [`DOCUMENTACION.md`](DOCUMENTACION.md#6-api-rest).

---

## 4. Ejecutar las pruebas

Requiere **Hurl** y **Docker iniciado**.

### Opción A: todo automático (recomendada)

```bash
make test-full
```

Baja cualquier contenedor previo, levanta la infraestructura, espera a la API, ejecuta las pruebas y, al terminar (con éxito o con error), limpia todo. Si alguna prueba falla, el comando termina con error.

### Opción B: con la aplicación ya levantada

```bash
make up
make wait
make test
make down
```

### Ejecutar Hurl directamente

```bash
hurl --test --variable base_url=http://localhost:8080 requests.hurl
```

Opciones útiles:

- `--very-verbose`: muestra requests y respuestas completas, para depurar.
- `--report-html reporte/`: genera un reporte HTML.

### Probar contra otra URL

```bash
make test BASE_URL=http://localhost:9090
```

### Pruebas unitarias de Go

```bash
make test-unit
```

---

## 5. Comandos del Makefile

Ejecutá `make help` para ver la lista.

| Comando | Descripción |
|---|---|
| `make up` | Levanta PostgreSQL y la API en segundo plano |
| `make down` | Detiene los contenedores y borra los volúmenes |
| `make restart` | Equivale a `down` + `up` |
| `make logs` | Muestra los logs en tiempo real |
| `make wait` | Espera (hasta 60 s) a que la API responda |
| `make test` | Ejecuta las pruebas Hurl (requiere la API levantada) |
| `make test-full` | Ciclo completo: levanta, prueba y limpia |
| `make test-unit` | Ejecuta `go test -v ./...` |
| `make run-local` | Levanta solo la base en Docker y ejecuta la API con `go run` |
| `make gen` | Ejecuta `sqlc generate` y `templ generate` |
| `make sqlc-gen` | Regenera el código de acceso a datos |
| `make templ-gen` | Regenera las vistas HTML |
| `make build` | Reconstruye las imágenes Docker |
| `make clean` | Alias de `down` |
| `make all` | Genera código y levanta la app en Docker |

---

## 6. Ejecución local de la API (sin contenedor)

Requiere **Go** instalado y Docker iniciado.

```bash
make run-local
```

Levanta solo la base de datos en Docker, espera a que esté lista y ejecuta `go run main.go` en tu máquina. La API queda disponible en <http://localhost:8080>. Para detenerla, presioná `Ctrl+C`; luego `make down` para apagar la base.

Si modificaste consultas SQL o plantillas, regenerá el código antes (requiere `sqlc` y `templ`):

```bash
make gen
```

---

## 7. Solución de problemas

| Síntoma | Causa probable | Solución |
|---|---|---|
| `Cannot connect to the Docker daemon` | Docker no está iniciado | Abrí Docker Desktop (o `sudo systemctl start docker`) y reintentá |
| `make: command not found` | `make` no está instalado | Instalalo (`sudo apt install make`, `brew install make`) o usá WSL en Windows |
| `hurl: command not found` | Hurl no está instalado | Ver [Requisitos previos](#1-requisitos-previos) |
| `make wait` termina con "la API no respondió a tiempo" | La API no pudo arrancar, a menudo por la base de datos | Revisá `make logs`; probá `make down` y luego `make up` |
| `port is already allocated` (8080 o 5432) | Otro programa usa ese puerto | Cerrá ese programa o cambiá el puerto publicado en `docker-compose.yml` |
| El contenedor `go-api` aparece como `Exited` | La API no pudo conectarse a la base al arrancar | `docker compose logs api`; probá `make down` y `make up` |
| Un test de Hurl falla con `invalid JSON` al leer el archivo | Un body que empieza con `{` se valida como JSON | Escribilo entre backticks (`` `...` ``) para enviarlo sin validar |
| Las pruebas fallan por datos de una corrida anterior | Quedaron registros en la base | `make down` (borra el volumen) y volvé a correr |
| Cambié `schema.sql` o `seed.sql` y no se nota | Esos scripts solo corren con el volumen vacío | `make down` y luego `make up` |
| Cambié el CSS y no se ve | La carpeta `static/` va dentro de la imagen, o caché del navegador | `make up` (reconstruye) y `Ctrl+Shift+R` en el navegador |
| Error al levantar por `seed.sql` | El archivo no existe o no se montó | Verificá que exista `sql/schema/seed.sql` |
| Al votar, error `500` con `violates check constraint` | Puntuación fuera del rango 1–5 | Enviá valores entre 1 y 5 |
