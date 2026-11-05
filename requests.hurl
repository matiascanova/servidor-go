# [VISTAS HTML] Carga de página principal
GET {{base_url}}/
HTTP 200
[Asserts]
header "Content-Type" contains "text/html"
xpath "string(//title)" contains "Foro"
xpath "count(//article[@class='entity-card'])" >= 1

# [VISTAS HTML] Validación de ruta inexistente (404)
GET {{base_url}}/no-existe
HTTP 404

# [VISTAS HTML] Renderizado de formulario de calificación para materia 1
GET {{base_url}}/materias/calificar?id=1
HTTP 200
[Asserts]
header "Content-Type" contains "text/html"
xpath "string(//section[@class='form-section']/h2)" contains "Agregar reseña"
xpath "string(//form/@action)" == "/votos"
xpath "string(//form/@method)" == "POST"
xpath "string(//input[@name='idmateria']/@value)" == "1"

# [VISTAS HTML] Validación de ID inválido en formulario (400)
GET {{base_url}}/materias/calificar?id=abc
HTTP 400

# [VISTAS HTML] Validación de ID inexistente en formulario (404)
GET {{base_url}}/materias/calificar?id=9999
HTTP 404

# [VISTAS HTML] Consulta de reseñas para materia (1)
GET {{base_url}}/materias/resenas?id=1
HTTP 200

# [VISTAS HTML] Consulta de reseñas para materia inexistente (404)
GET {{base_url}}/materias/resenas?id=9999
HTTP 404

# [FORMULARIOS] Envío de voto y redirección a reseñas de la materia (303)
POST {{base_url}}/votos
[FormParams]
idmateria: 1
puntuacion_dif: 4
puntuacion_rel: 5
puntuacion_gusto: 3
descripcion: Muy buena cursada (form)
HTTP 303
[Asserts]
header "Location" == "/materias/resenas?id=1"

# [VISTAS HTML] Confirmación de voto publicado en la vista de reseñas
GET {{base_url}}/materias/resenas?id=1
HTTP 200
[Asserts]
body contains "Muy buena cursada (form)"
body contains "4/5"
xpath "count(//article[@class='voto-card'])" == 1

# [VISTAS HTML] Modo edición precargado en formulario tras votar
GET {{base_url}}/materias/calificar?id=1
HTTP 200
[Asserts]
xpath "string(//section[@class='form-section']/h2)" contains "Editar tu reseña"
xpath "string(//input[@name='puntuacion_dif']/@value)" == "4"
xpath "string(//input[@name='puntuacion_rel']/@value)" == "5"
xpath "string(//textarea[@name='descripcion'])" == "Muy buena cursada (form)"

# [FORMULARIOS] Rechazo de puntuación fuera de rango 1-5 (400)
POST {{base_url}}/votos
[FormParams]
idmateria: 1
puntuacion_dif: 9
puntuacion_rel: 5
puntuacion_gusto: 3
descripcion: inválido
HTTP 400

# [FORMULARIOS] Rechazo de valores no numéricos en campos de puntuación (400)
POST {{base_url}}/votos
[FormParams]
idmateria: 1
puntuacion_dif: abc
puntuacion_rel: 5
puntuacion_gusto: 3
HTTP 400
