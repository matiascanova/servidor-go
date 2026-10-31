
# SECCIÓN 1: USUARIOS

# 1. Crear usuario
POST {{base_url}}/usuarios
Content-Type: application/json
{
  "idusuario": 10,
  "nombre": "Carlos",
  "apellido": "Gómez"
}
HTTP 201
[Asserts]
header "Content-Type" contains "application/json"
header "Location" == "/usuarios/10"
jsonpath "$.idusuario" == 10
jsonpath "$.nombre" == "Carlos"
jsonpath "$.apellido" == "Gómez"

# 2. Crear usuario duplicado -> 409
POST {{base_url}}/usuarios
Content-Type: application/json
{
  "idusuario": 10,
  "nombre": "Otro",
  "apellido": "Usuario"
}
HTTP 409
[Asserts]
jsonpath "$.error" == "el recurso ya existe"

# 3. Listar usuarios
GET {{base_url}}/usuarios
HTTP 200
[Asserts]
jsonpath "$[*].nombre" contains "Carlos"
jsonpath "$[*].idusuario" contains 1

# 4. Obtener usuario por ID
GET {{base_url}}/usuarios/10
HTTP 200
[Asserts]
jsonpath "$.nombre" == "Carlos"

# 5. Actualizar usuario
PUT {{base_url}}/usuarios/10
Content-Type: application/json
{
  "nombre": "Carlos Alberto",
  "apellido": "Gómez"
}
HTTP 200
[Asserts]
jsonpath "$.nombre" == "Carlos Alberto"

# 6. Verificar que la actualización persistió
GET {{base_url}}/usuarios/10
HTTP 200
[Asserts]
jsonpath "$.nombre" == "Carlos Alberto"

# --- Errores de usuarios ---

# Usuario inexistente
GET {{base_url}}/usuarios/9999
HTTP 404
[Asserts]
jsonpath "$.error" contains "no encontrado"

# Actualizar usuario inexistente
PUT {{base_url}}/usuarios/9999
Content-Type: application/json
{
  "nombre": "X",
  "apellido": "Y"
}
HTTP 404

# ID inválido
GET {{base_url}}/usuarios/abc
HTTP 400
[Asserts]
jsonpath "$.error" contains "inválido"

# JSON inválido
POST {{base_url}}/usuarios
Content-Type: application/json
`{ esto no es json }`
HTTP 400
[Asserts]
jsonpath "$.error" == "JSON inválido"

# Falta nombre
POST {{base_url}}/usuarios
Content-Type: application/json
{
  "idusuario": 11,
  "nombre": "",
  "apellido": "Sin Nombre"
}
HTTP 400
[Asserts]
jsonpath "$.error" contains "obligatorios"

# Método no permitido
DELETE {{base_url}}/usuarios
HTTP 405

# =============================================================================
# SECCIÓN 2: MATERIAS
# =============================================================================

# 7. Crear materia
POST {{base_url}}/materias
Content-Type: application/json
{
  "idmateria": 101,
  "nombre": "Programación Web",
  "cuatrimestre": 1,
  "descripcion": "Materia de Go y Docker"
}
HTTP 201
[Asserts]
header "Location" == "/materias/101"
jsonpath "$.idmateria" == 101
jsonpath "$.nombre" == "Programación Web"
jsonpath "$.cuatrimestre" == 1
jsonpath "$.descripcion" == "Materia de Go y Docker"

# 8. Materia sin descripción -> descripcion es null (no {"String":"","Valid":false})
POST {{base_url}}/materias
Content-Type: application/json
{
  "idmateria": 102,
  "nombre": "Materia sin descripción",
  "cuatrimestre": 3
}
HTTP 201
[Asserts]
jsonpath "$.descripcion" == null

# 9. Listar materias (JSON de dominio, sin estructuras nullable)
GET {{base_url}}/materias
HTTP 200
[Asserts]
jsonpath "$[*].nombre" contains "Programación Web"
jsonpath "$[*].descripcion" contains "Materia de Go y Docker"
body not contains "\"Valid\""
body not contains "\"String\""

# 10. Obtener materia por ID
GET {{base_url}}/materias/101
HTTP 200
[Asserts]
jsonpath "$.nombre" == "Programación Web"

# 11. Actualizar materia
PUT {{base_url}}/materias/101
Content-Type: application/json
{
  "nombre": "Programación Web II",
  "cuatrimestre": 2,
  "descripcion": "Materia de Go, Docker y Kubernetes"
}
HTTP 200
[Asserts]
jsonpath "$.nombre" == "Programación Web II"
jsonpath "$.cuatrimestre" == 2

# 12. Verificar que el PUT realmente persistió
GET {{base_url}}/materias/101
HTTP 200
[Asserts]
jsonpath "$.nombre" == "Programación Web II"
jsonpath "$.cuatrimestre" == 2
jsonpath "$.descripcion" == "Materia de Go, Docker y Kubernetes"

# 13. Eliminar materia auxiliar
DELETE {{base_url}}/materias/102
HTTP 204

GET {{base_url}}/materias/102
HTTP 404

# --- Errores de materias ---

GET {{base_url}}/materias/9999
HTTP 404
[Asserts]
jsonpath "$.error" == "Materia no encontrada"

PUT {{base_url}}/materias/9999
Content-Type: application/json
{
  "nombre": "Inexistente",
  "cuatrimestre": 1
}
HTTP 404

GET {{base_url}}/materias/abc
HTTP 400

# Cuatrimestre inválido
POST {{base_url}}/materias
Content-Type: application/json
{
  "idmateria": 103,
  "nombre": "Materia inválida",
  "cuatrimestre": 0
}
HTTP 400
[Asserts]
jsonpath "$.error" contains "cuatrimestre"

# Materia duplicada
POST {{base_url}}/materias
Content-Type: application/json
{
  "idmateria": 101,
  "nombre": "Duplicada",
  "cuatrimestre": 1
}
HTTP 409

# =============================================================================
# SECCIÓN 3: VISTAS HTML (templ)
# =============================================================================

# 14. Página principal: lista de materias renderizada en el servidor
GET {{base_url}}/
HTTP 200
[Asserts]
header "Content-Type" contains "text/html"
xpath "string(//title)" contains "Foro"
xpath "count(//article[@class='entity-card'])" >= 1
body contains "Programación Web II"
body contains "/materias/calificar?id=101"
body contains "/materias/resenas?id=101"

# Ruta inexistente -> 404 (GET /{$} solo coincide con la raíz)
GET {{base_url}}/no-existe
HTTP 404

# 15. Formulario de calificación vacío (sin voto previo del usuario demo)
GET {{base_url}}/materias/calificar?id=101
HTTP 200
[Asserts]
header "Content-Type" contains "text/html"
xpath "string(//section[@class='form-section']/h2)" contains "Agregar reseña"
xpath "string(//form/@action)" == "/votos"
xpath "string(//form/@method)" == "POST"
xpath "string(//input[@name='idmateria']/@value)" == "101"

GET {{base_url}}/materias/calificar?id=abc
HTTP 400

GET {{base_url}}/materias/calificar?id=9999
HTTP 404

# 16. Reseñas vacías
GET {{base_url}}/materias/resenas?id=101
HTTP 200
[Asserts]
body contains "Aún no hay reseñas"

GET {{base_url}}/materias/resenas?id=9999
HTTP 404

# 17. Enviar el formulario (PRG): 303 + Location "/"
POST {{base_url}}/votos
[Form]
idmateria: 101
puntuacion_dif: 4
puntuacion_rel: 5
puntuacion_gusto: 3
descripcion: Muy buena cursada (form)
HTTP 303
[Asserts]
header "Location" == "/"

# 18. El voto cargado por formulario aparece en las reseñas
GET {{base_url}}/materias/resenas?id=101
HTTP 200
[Asserts]
body contains "Muy buena cursada (form)"
body contains "4/5"
xpath "count(//article[@class='voto-card'])" == 1

# 19. El formulario ahora está en modo edición y precargado
GET {{base_url}}/materias/calificar?id=101
HTTP 200
[Asserts]
xpath "string(//section[@class='form-section']/h2)" contains "Editar tu reseña"
xpath "string(//input[@name='puntuacion_dif']/@value)" == "4"
xpath "string(//input[@name='puntuacion_rel']/@value)" == "5"
xpath "string(//textarea[@name='descripcion'])" == "Muy buena cursada (form)"

# 20. Formulario con puntuación fuera de rango -> 400 (no se guarda)
POST {{base_url}}/votos
[Form]
idmateria: 101
puntuacion_dif: 9
puntuacion_rel: 5
puntuacion_gusto: 3
descripcion: inválido
HTTP 400

# 21. Formulario con valores no numéricos -> 400
POST {{base_url}}/votos
[Form]
idmateria: 101
puntuacion_dif: abc
puntuacion_rel: 5
puntuacion_gusto: 3
HTTP 400

# 22. Limpieza del voto del formulario
DELETE {{base_url}}/votos/1/101
HTTP 204

# =============================================================================
# SECCIÓN 4: VOTOS (JSON) - usuario 10 / materia 101
# =============================================================================

# 23. Crear voto (el Content-Type puede traer charset)
POST {{base_url}}/votos
Content-Type: application/json; charset=utf-8
{
  "idusuario": 10,
  "idmateria": 101,
  "puntuacion_dif": 3,
  "puntuacion_rel": 4,
  "puntuacion_gusto": 5,
  "descripcion": "Excelente materia"
}
HTTP 201
[Asserts]
header "Location" == "/votos/10/101"
jsonpath "$.idusuario" == 10
jsonpath "$.idmateria" == 101
jsonpath "$.puntuacion_dif" == 3
jsonpath "$.puntuacion_rel" == 4
jsonpath "$.puntuacion_gusto" == 5
jsonpath "$.descripcion" == "Excelente materia"

# 24. Crear voto duplicado -> 409
POST {{base_url}}/votos
Content-Type: application/json
{
  "idusuario": 10,
  "idmateria": 101,
  "puntuacion_dif": 1
}
HTTP 409

# 25. Listar votos de una materia (idmateria es obligatorio)
GET {{base_url}}/votos
HTTP 400
[Asserts]
jsonpath "$.error" contains "idmateria"

GET {{base_url}}/votos?idmateria=abc
HTTP 400

# 26. Listar votos filtrando por materia (JSON de dominio, sin nullables)
GET {{base_url}}/votos?idmateria=101
HTTP 200
[Asserts]
jsonpath "$" count == 1
jsonpath "$[0].idusuario" == 10
jsonpath "$[0].descripcion" == "Excelente materia"
body not contains "\"Valid\""

# Materia sin votos -> lista vacía (y no null)
GET {{base_url}}/votos?idmateria=9999
HTTP 200
[Asserts]
jsonpath "$" count == 0

# 27. Obtener voto por ID compuesto
GET {{base_url}}/votos/10/101
HTTP 200
[Asserts]
jsonpath "$.puntuacion_gusto" == 5
jsonpath "$.descripcion" == "Excelente materia"

# 28. Actualizar voto (PUT reemplaza: puntuacion_rel y puntuacion_gusto quedan en null)
PUT {{base_url}}/votos/10/101
Content-Type: application/json
{
  "puntuacion_dif": 2,
  "descripcion": "Opinión actualizada de la materia"
}
HTTP 200
[Asserts]
jsonpath "$.puntuacion_dif" == 2
jsonpath "$.puntuacion_rel" == null
jsonpath "$.descripcion" == "Opinión actualizada de la materia"

# 29. Verificar que el PUT persistió
GET {{base_url}}/votos/10/101
HTTP 200
[Asserts]
jsonpath "$.puntuacion_dif" == 2
jsonpath "$.puntuacion_rel" == null
jsonpath "$.puntuacion_gusto" == null
jsonpath "$.descripcion" == "Opinión actualizada de la materia"

# --- Errores de votos ---

# Puntuación fuera de rango
POST {{base_url}}/votos
Content-Type: application/json
{
  "idusuario": 10,
  "idmateria": 101,
  "puntuacion_dif": 6
}
HTTP 400
[Asserts]
jsonpath "$.error" contains "entre 1 y 5"

# Sin ninguna puntuación
PUT {{base_url}}/votos/10/101
Content-Type: application/json
{
  "descripcion": "sin puntuaciones"
}
HTTP 400
[Asserts]
jsonpath "$.error" contains "al menos una puntuación"

# Usuario inexistente (violación de FK)
POST {{base_url}}/votos
Content-Type: application/json
{
  "idusuario": 9999,
  "idmateria": 101,
  "puntuacion_dif": 3
}
HTTP 400
[Asserts]
jsonpath "$.error" contains "inexistente"

# JSON inválido
POST {{base_url}}/votos
Content-Type: application/json
`{ no es json }`
HTTP 400
[Asserts]
jsonpath "$.error" == "JSON inválido"

# Actualizar voto inexistente
PUT {{base_url}}/votos/10/999
Content-Type: application/json
{
  "puntuacion_dif": 3
}
HTTP 404

# IDs inválidos
GET {{base_url}}/votos/abc/101
HTTP 400

# Ruta sin ID compuesto
GET {{base_url}}/votos/10
HTTP 404

# =============================================================================
# SECCIÓN 5: ELIMINACIÓN Y LIMPIEZA
# =============================================================================

# 30. Eliminar voto
DELETE {{base_url}}/votos/10/101
HTTP 204

# 31. Verificar eliminación del voto
GET {{base_url}}/votos/10/101
HTTP 404
[Asserts]
jsonpath "$.error" contains "no encontrado"

# 32. Eliminar materia y usuario de prueba
DELETE {{base_url}}/materias/101
HTTP 204

GET {{base_url}}/materias/101
HTTP 404

DELETE {{base_url}}/usuarios/10
HTTP 204

GET {{base_url}}/usuarios/10
HTTP 404
