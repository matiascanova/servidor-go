# Foro de Alumnos - Exactas UNICEN

## Requisitos

- Go 1.26 o superior
- Docker con Docker Compose
- make
- templ
- Hurl (solo para las pruebas)

## Comandos

```bash
make        # genera el código, levanta la base de datos y corre la app en http://localhost:8080
make test   # corre las pruebas (con la app ya corriendo, en otra terminal)
make down   # detiene la base de datos y borra sus datos
```

## Cuentas y sesiones

- Cualquiera (invitado) puede ver las materias y las reseñas de cada una.
- Para **calificar**, ver **Mis reseñas** o borrar una reseña hay que tener cuenta:
  `/registro` crea la cuenta (queda logueado) y `/login` / `/logout` abren y cierran sesión.
- Las contraseñas se guardan con hash (PBKDF2-HMAC-SHA256 con sal por usuario), nunca en claro.
- La sesión es una cookie `HttpOnly` + `SameSite=Lax` de 7 días. En la base se guarda solo el hash
  del token (tabla `sesion`).
- Las reseñas se asignan siempre al usuario de la sesión, por lo que nadie puede ver ni borrar las ajenas.
- Si la app se sirve por HTTPS, definir `COOKIE_SECURE=true` para marcar la cookie como `Secure`.

## Datos iniciales

`docker-compose.yml` ejecuta, en orden, `sql/schema/schema.sql`, `sql/schema/seed.sql` (materias y
el usuario demo) y `sql/schema/post_seed.sql` (sincroniza la secuencia de ids de usuario con lo que
cargó el seed). El usuario demo del seed no tiene email ni contraseña, así que no puede iniciar
sesión: las cuentas reales se crean desde `/registro`.

## Pruebas

`make test` ejecuta `requests.hurl`, que crea sus propias cuentas con un `run_id` distinto en cada
corrida. Para correr Hurl a mano hay que pasar esa variable:
 make test debe ejecutarse desde una nueva terminal
