# requests.hurl - Pruebas de integración HTTP para Usuarios, Materias y Votos (TP3)
#
# Uso:
#   hurl --test --variable base_url=http://localhost:8080 requests.hurl
# o bien:
#   make test
#
# Las entradas se ejecutan en orden y comparten estado en la BD, igual que el script bash.

# SECCIÓN 1: USUARIOS (/usuarios)

# 1. Crear usuario
POST {{base_url}}/usuarios
Content-Type: application/json
{
  "idusuario": 10,
  "nombre": "Carlos",
  "apellido": "Gómez",
  "curso": true
}
HTTP 201
[Asserts]
body contains "Carlos"

# 2. Listar usuarios
GET {{base_url}}/usuarios
HTTP 200
[Asserts]
body contains "Carlos"

# 3. Obtener usuario por ID
GET {{base_url}}/usuarios/10
HTTP 200
[Asserts]
body contains "Carlos"

# 4. Actualizar usuario
PUT {{base_url}}/usuarios/10
Content-Type: application/json
{
  "nombre": "Carlos Alberto",
  "apellido": "Gómez",
  "curso": false
}
HTTP 200
[Asserts]
body contains "Carlos Alberto"

# SECCIÓN 2: MATERIAS (/materias)

# 5. Crear materia
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
body contains "Programación Web"

# 6. Listar materias
GET {{base_url}}/materias
HTTP 200
[Asserts]
body contains "Programación Web"

# 7. Obtener materia por ID
GET {{base_url}}/materias/101
HTTP 200
[Asserts]
body contains "Programación Web"

# 8. Actualizar materia
PUT {{base_url}}/materias/101
Content-Type: application/json
{
  "nombre": "Programación Web II",
  "cuatrimestre": 2,
  "descripcion": "Materia de Go, Docker y Kubernetes"
}
HTTP 200
[Asserts]
body contains "Programación Web II"

# SECCIÓN 3: VOTOS (/votos) - relaciona Usuario 10 y Materia 101

# 9. Crear voto
# (el Content-Type debe ser exactamente application/json: el dispatcher lo compara por igualdad)
POST {{base_url}}/votos
Content-Type: application/json
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
body contains "exitosa"

# 10. Obtener voto por ID compuesto
GET {{base_url}}/votos/10/101
HTTP 200
[Asserts]
body contains "Excelente materia"

# 11. Actualizar voto
PUT {{base_url}}/votos/10/101
Content-Type: application/json
{
  "puntuacion_dif": 2,
  "puntuacion_rel": 3,
  "puntuacion_gusto": 4,
  "descripcion": "Opinión actualizada de la materia"
}
HTTP 200
[Asserts]
body contains "Opinión actualizada"

# 12. Eliminar voto
DELETE {{base_url}}/votos/10/101
HTTP 204

# 13. Verificar eliminación del voto (404)
GET {{base_url}}/votos/10/101
HTTP 404
[Asserts]
body contains "no encontrado"

# SECCIÓN 4: ELIMINACIÓN Y LIMPIEZA

# 14. Eliminar materia
DELETE {{base_url}}/materias/101
HTTP 204

# 15. Eliminar usuario
DELETE {{base_url}}/usuarios/10
HTTP 204

# test de errores

# Usuario inexistente
GET {{base_url}}/usuarios/9999
HTTP 404

# ID inválido
GET {{base_url}}/usuarios/abc
HTTP 400

# JSON inválido
POST {{base_url}}/usuarios
Content-Type: application/json
`{ esto no es json }`
HTTP 400
[Asserts]
body contains "JSON inválido"
