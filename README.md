# Foro de Alumnos - Exactas UNICEN

## Requisitos

- Go 1.22 o superior
- Docker con Docker Compose
- make
- [templ](https://templ.guide) y [sqlc](https://sqlc.dev):
  ```bash
  go install github.com/a-h/templ/cmd/templ@v0.2.778
  go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
  ```
- [Hurl](https://hurl.dev) (solo para las pruebas)

## Comandos

```bash
make up       # genera el código, levanta la base de datos y corre la app en http://localhost:8080
make test   # corre las pruebas (con la app ya corriendo, en otra terminal)
make down   # detiene la base de datos y borra sus datos
```
