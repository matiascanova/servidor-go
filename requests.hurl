# =============================================================================
# Tests de la app web (templ + HTMX + autenticación) - Foro de alumnos
#
# Uso (con la app levantada en :8080 y la BD con datos iniciales):
#   hurl --test --variable base_url=http://localhost:8080 \
#        --variable run_id=$(date +%s) requests.hurl
#   (o simplemente: make test, que ya pasa run_id)
#
# Supuestos sobre los datos iniciales (seed.sql):
#   - existe al menos una materia (se toma la primera de la home)
# Ya no hace falta un usuario precargado: las pruebas crean sus propias cuentas
# (emails con run_id, así que se pueden repetir las corridas) y Hurl conserva la
# cookie de sesión entre pedidos del mismo archivo.
# =============================================================================

# =============================================================================
# SECCIÓN 0: PREPARACIÓN (como invitado)
# =============================================================================

# Home: captura el id y el nombre de la primera materia
GET {{base_url}}/
HTTP 200
[Captures]
materia_id: regex "calificar\\?id=(\\d+)"
materia_nombre: xpath "string((//article[@class='entity-card'])[1]/div[@class='subject-title'])"
[Asserts]
# Un invitado ve las opciones de ingreso y no la de salir
body contains "Ingresar"
body contains "Registrarse"
body not contains "Salir"

# =============================================================================
# SECCIÓN A: INVITADO (sin sesión)
# =============================================================================

# Las reseñas de una materia son públicas
GET {{base_url}}/materias/resenas?id={{materia_id}}
HTTP 200

# Páginas protegidas: redirigen al login recordando a dónde se quería ir
GET {{base_url}}/materias/calificar?id={{materia_id}}
HTTP 303
[Asserts]
header "Location" contains "/login?next="

GET {{base_url}}/mis-resenas
HTTP 303
[Asserts]
header "Location" contains "/login"

# Acciones protegidas: 401 (y HX-Redirect si vienen de HTMX)
POST {{base_url}}/votos
HX-Request: true
[Form]
idmateria: {{materia_id}}
puntuacion_dif: 4
HTTP 401
[Asserts]
header "HX-Redirect" == "/login"

POST {{base_url}}/votos
[Form]
idmateria: {{materia_id}}
puntuacion_dif: 4
HTTP 401

DELETE {{base_url}}/mis-resenas/{{materia_id}}
HTTP 401

DELETE {{base_url}}/votos/eliminar
[QueryStringParams]
idmateria: {{materia_id}}
HTTP 401

# Formularios de acceso
GET {{base_url}}/login
HTTP 200
[Asserts]
xpath "string(//form/@action)" == "/login"
xpath "count(//input[@name='email'])" == 1
xpath "count(//input[@name='password'])" == 1

GET {{base_url}}/registro
HTTP 200
[Asserts]
xpath "string(//form/@action)" == "/registro"
xpath "count(//input[@name='password2'])" == 1

# =============================================================================
# SECCIÓN B: REGISTRO (validaciones y alta)
# =============================================================================

# Contraseñas distintas
POST {{base_url}}/registro
[Form]
nombre: Test
apellido: Usuario
email: test-{{run_id}}@example.com
password: clave-segura-123
password2: otra-clave-distinta
HTTP 400
[Asserts]
body contains "no coinciden"
# Se conserva lo escrito (menos la contraseña)
xpath "string(//input[@name='email']/@value)" == "test-{{run_id}}@example.com"

# Contraseña demasiado corta
POST {{base_url}}/registro
[Form]
nombre: Test
apellido: Usuario
email: test-{{run_id}}@example.com
password: corta
password2: corta
HTTP 400
[Asserts]
body contains "entre 8"

# Email inválido
POST {{base_url}}/registro
[Form]
nombre: Test
apellido: Usuario
email: esto-no-es-un-email
password: clave-segura-123
password2: clave-segura-123
HTTP 400
[Asserts]
body contains "no es válido"

# Falta el nombre
POST {{base_url}}/registro
[Form]
apellido: Usuario
email: test-{{run_id}}@example.com
password: clave-segura-123
password2: clave-segura-123
HTTP 400
[Asserts]
body contains "obligatorios"

# Alta correcta: redirige a la home y deja la cookie de sesión
POST {{base_url}}/registro
[Form]
nombre: Test
apellido: Usuario
email: test-{{run_id}}@example.com
password: clave-segura-123
password2: clave-segura-123
HTTP 303
[Asserts]
header "Location" == "/"
header "Set-Cookie" contains "session="
header "Set-Cookie" contains "HttpOnly"
header "Set-Cookie" contains "SameSite=Lax"

# Ya con sesión: la home muestra el nombre y "Salir"
GET {{base_url}}/
HTTP 200
[Asserts]
body contains "Hola, Test"
body contains "Salir"
body not contains "Registrarse"

# Con sesión, /login y /registro redirigen
GET {{base_url}}/login
HTTP 303
[Asserts]
header "Location" == "/"

GET {{base_url}}/registro
HTTP 303

# Email repetido -> 409 (se normaliza a minúsculas, así que cambiar mayúsculas no sirve)
POST {{base_url}}/registro
[Form]
nombre: Test
apellido: Usuario
email: TEST-{{run_id}}@EXAMPLE.COM
password: clave-segura-123
password2: clave-segura-123
HTTP 409
[Asserts]
body contains "Ya existe una cuenta"

# =============================================================================
# SECCIÓN C: LOGOUT Y LOGIN
# =============================================================================

POST {{base_url}}/logout
HTTP 303
[Asserts]
header "Location" == "/"

# Sin sesión otra vez
GET {{base_url}}/mis-resenas
HTTP 303
[Asserts]
header "Location" contains "/login"

# Contraseña incorrecta
POST {{base_url}}/login
[Form]
email: test-{{run_id}}@example.com
password: clave-equivocada
HTTP 401
[Asserts]
body contains "Email o contraseña incorrectos"
header "Set-Cookie" not exists

# Email inexistente: mismo mensaje (no revela qué cuentas existen)
POST {{base_url}}/login
[Form]
email: no-existe-{{run_id}}@example.com
password: clave-segura-123
HTTP 401
[Asserts]
body contains "Email o contraseña incorrectos"
header "Set-Cookie" not exists

# Login correcto con un "next" malicioso: se ignora (evita open redirect)
POST {{base_url}}/login
[Form]
email: test-{{run_id}}@example.com
password: clave-segura-123
next: //sitio-malicioso.example.com
HTTP 303
[Asserts]
header "Location" == "/"
header "Set-Cookie" contains "session="

# Con sesión, el "next" local sí se respeta
GET {{base_url}}/login?next=/mis-resenas
HTTP 303
[Asserts]
header "Location" == "/mis-resenas"

# Se puede entrar a lo protegido
GET {{base_url}}/mis-resenas
HTTP 200

# =============================================================================
# SECCIÓN 0 (original): PREPARACIÓN, ya con sesión iniciada
# =============================================================================

# Estado limpio: borra la reseña del usuario para esa materia
# (DeleteVoto no falla si no existía, así que es idempotente)
DELETE {{base_url}}/mis-resenas/{{materia_id}}
HTTP 200

# =============================================================================
# SECCIÓN 1: PÁGINA PRINCIPAL
# =============================================================================

GET {{base_url}}/
HTTP 200
[Asserts]
header "Content-Type" contains "text/html"
xpath "string(//title)" contains "Foro"
xpath "count(//div[@id='entities-container'])" == 1
xpath "count(//article[@class='entity-card'])" >= 1
# HTMX cargado desde el layout (punto 1 de la consigna)
xpath "count(//script[contains(@src,'htmx')])" >= 1
body contains "/materias/calificar?id={{materia_id}}"
body contains "/materias/resenas?id={{materia_id}}"

# Ruta inexistente -> 404 (GET /{$} solo coincide con la raíz)
GET {{base_url}}/no-existe
HTTP 404

# Método no permitido sobre la raíz
DELETE {{base_url}}/
HTTP 405

# Los endpoints solo aceptan los métodos registrados
GET {{base_url}}/votos
HTTP 405

# Estáticos
GET {{base_url}}/static/styles.css
HTTP 200
[Asserts]
header "Content-Type" contains "text/css"

# =============================================================================
# SECCIÓN 2: FORMULARIO DE CALIFICACIÓN (sin voto previo)
# =============================================================================

GET {{base_url}}/materias/calificar?id={{materia_id}}
HTTP 200
[Asserts]
header "Content-Type" contains "text/html"
xpath "string(//section[@id='form-container']/h2)" contains "Agregar reseña"
xpath "string(//section[@id='form-container']/h2)" contains "{{materia_nombre}}"
# El formulario es reactivo (HTMX), no un form clásico con action/method
xpath "string(//form[@hx-post]/@hx-post)" == "/votos"
xpath "string(//form[@hx-post]/@hx-target)" == "#form-container"
xpath "string(//form[@hx-post]/@hx-swap)" == "outerHTML"
xpath "string(//input[@name='idmateria']/@value)" == "{{materia_id}}"
xpath "count(//input[@type='number'])" == 3
# Sin voto previo no hay botón de eliminar
xpath "count(//button[@class='btn-delete'])" == 0

GET {{base_url}}/materias/calificar?id=abc
HTTP 400

GET {{base_url}}/materias/calificar?id=0
HTTP 400

GET {{base_url}}/materias/calificar?id=999999
HTTP 404

GET {{base_url}}/materias/resenas?id=abc
HTTP 400

GET {{base_url}}/materias/resenas?id=999999
HTTP 404

# =============================================================================
# SECCIÓN 3: CREAR RESEÑA CON HTMX (POST devuelve fragmento, sin redirección)
# =============================================================================

# 3.1 Alta: responde 200 (no 303) con solo el fragmento del formulario
POST {{base_url}}/votos
HX-Request: true
[Form]
idmateria: {{materia_id}}
puntuacion_dif: 4
puntuacion_rel: 5
puntuacion_gusto: 3
descripcion: Muy buena cursada (form)
HTTP 200
[Asserts]
header "Content-Type" contains "text/html"
header "Location" not exists
body not contains "<html"
xpath "string(//section[@id='form-container']/h2)" contains "Editar tu reseña"
xpath "string(//input[@name='puntuacion_dif']/@value)" == "4"
xpath "string(//input[@name='puntuacion_rel']/@value)" == "5"
xpath "string(//input[@name='puntuacion_gusto']/@value)" == "3"
xpath "string(//textarea[@name='descripcion'])" == "Muy buena cursada (form)"
# Ahora sí aparece el botón de eliminar, apuntando a la materia
xpath "string(//button[@class='btn-delete']/@hx-delete)" contains "idmateria={{materia_id}}"

# 3.2 La reseña aparece en la página de reseñas de la materia
GET {{base_url}}/materias/resenas?id={{materia_id}}
HTTP 200
[Asserts]
body contains "Muy buena cursada (form)"
body contains "4/5"
xpath "count(//article[@class='voto-card'])" >= 1

# 3.3 El formulario queda en modo edición y precargado
GET {{base_url}}/materias/calificar?id={{materia_id}}
HTTP 200
[Asserts]
xpath "string(//section[@id='form-container']/h2)" contains "Editar tu reseña"
xpath "string(//input[@name='puntuacion_dif']/@value)" == "4"
xpath "string(//input[@name='puntuacion_rel']/@value)" == "5"
xpath "string(//input[@name='puntuacion_gusto']/@value)" == "3"
xpath "string(//textarea[@name='descripcion'])" == "Muy buena cursada (form)"

# 3.4 Aparece en "Mis reseñas" con su botón de borrado HTMX
GET {{base_url}}/mis-resenas
HTTP 200
[Asserts]
header "Content-Type" contains "text/html"
xpath "count(//article[@class='mi-resena'])" >= 1
body contains "hx-delete=\"/mis-resenas/{{materia_id}}\""
body contains "Muy buena cursada (form)"

# =============================================================================
# SECCIÓN 4: ACTUALIZAR (el POST hace upsert; los campos omitidos quedan en NULL)
# =============================================================================

POST {{base_url}}/votos
HX-Request: true
[Form]
idmateria: {{materia_id}}
puntuacion_dif: 2
descripcion: Opinión actualizada
HTTP 200
[Asserts]
xpath "string(//section[@id='form-container']/h2)" contains "Editar tu reseña"
xpath "string(//input[@name='puntuacion_dif']/@value)" == "2"
xpath "string(//input[@name='puntuacion_rel']/@value)" == ""
xpath "string(//input[@name='puntuacion_gusto']/@value)" == ""
xpath "string(//textarea[@name='descripcion'])" == "Opinión actualizada"

# Persistió y reemplazó la reseña anterior (no se duplicó)
GET {{base_url}}/materias/resenas?id={{materia_id}}
HTTP 200
[Asserts]
body contains "Opinión actualizada"
body not contains "Muy buena cursada (form)"

# =============================================================================
# SECCIÓN 5: VALIDACIONES DEL FORMULARIO (400 y no se guarda)
# =============================================================================

# Puntuación fuera de rango (alta)
POST {{base_url}}/votos
[Form]
idmateria: {{materia_id}}
puntuacion_dif: 9
puntuacion_rel: 5
puntuacion_gusto: 3
descripcion: inválido
HTTP 400
[Asserts]
body contains "entre 1 y 5"

# Puntuación fuera de rango (baja)
POST {{base_url}}/votos
[Form]
idmateria: {{materia_id}}
puntuacion_gusto: 0
HTTP 400
[Asserts]
body contains "entre 1 y 5"

# Valores no numéricos
POST {{base_url}}/votos
[Form]
idmateria: {{materia_id}}
puntuacion_dif: abc
puntuacion_rel: 5
puntuacion_gusto: 3
HTTP 400
[Asserts]
body contains "valores numéricos incorrectos"

# Falta idmateria
POST {{base_url}}/votos
[Form]
puntuacion_dif: 3
HTTP 400

# Sin ninguna puntuación
POST {{base_url}}/votos
[Form]
idmateria: {{materia_id}}
descripcion: sin puntuaciones
HTTP 400
[Asserts]
body contains "al menos una puntuación"

# Descripción de más de 140 caracteres
POST {{base_url}}/votos
[Form]
idmateria: {{materia_id}}
puntuacion_dif: 3
descripcion: aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
HTTP 400
[Asserts]
body contains "140 caracteres"

# Materia inexistente (violación de FK)
POST {{base_url}}/votos
[Form]
idmateria: 999999
puntuacion_dif: 3
HTTP 400
[Asserts]
body contains "no es válido"

# Ninguno de los intentos inválidos modificó la reseña guardada
GET {{base_url}}/materias/calificar?id={{materia_id}}
HTTP 200
[Asserts]
xpath "string(//input[@name='puntuacion_dif']/@value)" == "2"
xpath "string(//textarea[@name='descripcion'])" == "Opinión actualizada"

# =============================================================================
# SECCIÓN 6: ELIMINAR DESDE EL FORMULARIO (DELETE /votos/eliminar)
# =============================================================================

# Responde 200 con el formulario de nuevo en modo "Agregar"
DELETE {{base_url}}/votos/eliminar
HX-Request: true
[QueryStringParams]
idmateria: {{materia_id}}
HTTP 200
[Asserts]
header "Content-Type" contains "text/html"
body not contains "<html"
xpath "string(//section[@id='form-container']/h2)" contains "Agregar reseña"
xpath "string(//input[@name='puntuacion_dif']/@value)" == ""
xpath "string(//textarea[@name='descripcion'])" == ""
xpath "count(//button[@class='btn-delete'])" == 0

# Se borró de verdad
GET {{base_url}}/materias/resenas?id={{materia_id}}
HTTP 200
[Asserts]
body not contains "Opinión actualizada"

# ID faltante o inválido
DELETE {{base_url}}/votos/eliminar
HTTP 400

DELETE {{base_url}}/votos/eliminar
[QueryStringParams]
idmateria: abc
HTTP 400

# =============================================================================
# SECCIÓN 7: ELIMINAR DESDE "MIS RESEÑAS" (DELETE /mis-resenas/{idmateria})
# =============================================================================

# Se vuelve a crear una reseña para borrarla desde esta vista
POST {{base_url}}/votos
HX-Request: true
[Form]
idmateria: {{materia_id}}
puntuacion_dif: 5
descripcion: Reseña para borrar
HTTP 200

GET {{base_url}}/mis-resenas
HTTP 200
[Asserts]
body contains "Reseña para borrar"

# Respuesta vacía con 200: HTMX reemplaza la tarjeta por nada
DELETE {{base_url}}/mis-resenas/{{materia_id}}
HX-Request: true
HTTP 200
[Asserts]
body == ""

# La reseña ya no existe ni en "Mis reseñas" ni en la materia
GET {{base_url}}/mis-resenas
HTTP 200
[Asserts]
body not contains "Reseña para borrar"
body not contains "hx-delete=\"/mis-resenas/{{materia_id}}\""

GET {{base_url}}/materias/resenas?id={{materia_id}}
HTTP 200
[Asserts]
body not contains "Reseña para borrar"

# Borrar algo ya borrado es idempotente
DELETE {{base_url}}/mis-resenas/{{materia_id}}
HTTP 200

# IDs inválidos
DELETE {{base_url}}/mis-resenas/abc
HTTP 400

DELETE {{base_url}}/mis-resenas/0
HTTP 400

# =============================================================================
# SECCIÓN 8: AISLAMIENTO ENTRE USUARIOS (nadie toca reseñas ajenas)
# =============================================================================

# El usuario A deja una reseña
POST {{base_url}}/votos
HX-Request: true
[Form]
idmateria: {{materia_id}}
puntuacion_dif: 5
descripcion: Reseña privada de A
HTTP 200

POST {{base_url}}/logout
HTTP 303

# Se registra el usuario B
POST {{base_url}}/registro
[Form]
nombre: Otro
apellido: Usuario
email: otro-{{run_id}}@example.com
password: clave-segura-456
password2: clave-segura-456
HTTP 303

# B no ve la reseña de A en "Mis reseñas" ni la hereda en el formulario
GET {{base_url}}/mis-resenas
HTTP 200
[Asserts]
body not contains "Reseña privada de A"

GET {{base_url}}/materias/calificar?id={{materia_id}}
HTTP 200
[Asserts]
xpath "string(//section[@id='form-container']/h2)" contains "Agregar reseña"
xpath "string(//input[@name='puntuacion_dif']/@value)" == ""

# B intenta borrar "su" reseña de esa materia: borra solo lo propio (nada)
DELETE {{base_url}}/mis-resenas/{{materia_id}}
HTTP 200

DELETE {{base_url}}/votos/eliminar
[QueryStringParams]
idmateria: {{materia_id}}
HTTP 200

# La reseña de A sigue existiendo
GET {{base_url}}/materias/resenas?id={{materia_id}}
HTTP 200
[Asserts]
body contains "Reseña privada de A"

POST {{base_url}}/logout
HTTP 303

# A vuelve a entrar y su reseña está intacta
POST {{base_url}}/login
[Form]
email: test-{{run_id}}@example.com
password: clave-segura-123
HTTP 303

GET {{base_url}}/mis-resenas
HTTP 200
[Asserts]
body contains "Reseña privada de A"

# Limpieza final
DELETE {{base_url}}/mis-resenas/{{materia_id}}
HTTP 200
