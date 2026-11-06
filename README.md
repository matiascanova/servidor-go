# Foro de Alumnos - Exactas UNICEN

## Requisitos

- Go 1.22 o superior
- Docker con Docker Compose
- make
- [templ](https://templ.guide) y [sqlc](https://sqlc.dev)
- [Hurl](https://hurl.dev) (solo para las pruebas)

## Comandos

```bash
make up       # genera el código, levanta la base de datos y corre la app en http://localhost:8080
make test   # corre las pruebas (con la app ya corriendo, en otra terminal)
make down   # detiene la base de datos y borra sus datos
```
## Nota extra:
- Es recomendable hacer el testeo primero antes de subir alguna reseña ya que si se sube una reseña y en una misma materia que requests testea no se va a anticipar esa nueva reseña y se va a generar un error
- Se subieron los go de templ generados en el ultimo testeo por redundancia en caso de que error, pero make los genera cuando se hace make up
