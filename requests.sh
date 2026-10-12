#!/usr/bin/env bash

set -e

BASE_URL="http://localhost:8080"
USUARIOS_URL="$BASE_URL/usuarios"
MATERIAS_URL="$BASE_URL/materias"
VOTOS_URL="$BASE_URL/votos"

if command -v sqlc > /dev/null 2>&1; then
  sqlc generate 
  echo "Código Go regenerado exitosamente en el paquete ./db" 
else 
  echo "Advertencia: 'sqlc' no se encuentra instalado en el sistema. Utilizando el código Go existente en ./db" 
fi

# Limpieza total al salir
cleanup() {
  echo " Limpiando contenedores y volúmenes Docker..."
  docker compose down -v 2>/dev/null || true
}
trap cleanup EXIT

echo " 1. Limpiando ejecuciones previas y volúmenes"
docker compose down -v 2>/dev/null || true


echo " 2. Levantando infraestructura con Docker Compose"
docker compose up -d --build

echo -e "\nEsperando a que la base de datos PostgreSQL esté lista..."
while ! docker exec postgres-db pg_isready -U postgres -d apirest > /dev/null 2>&1; do
    sleep 1
done
echo -e "\n Base de datos lista."

echo -e "\nEsperando a que la API REST responda en http://localhost:8080/..."
MAX_RETRIES=20
COUNT=0
until curl -s "$USUARIOS_URL" > /dev/null 2>&1 || [ $COUNT -eq $MAX_RETRIES ]; do
  sleep 1
  COUNT=$((COUNT + 1))
done

if [ $COUNT -eq $MAX_RETRIES ]; then
  echo "Error: La API no respondió a tiempo."
  echo "Mostrando logs de los contenedores:"
  docker compose logs
  exit 1
fi

echo "API REST en línea. Iniciando suite completa de pruebas HTTP..."


# SECCIÓN 1: PRUEBAS DE USUARIOS (/usuarios)

echo "--- [1/15] PRUEBAS DE USUARIOS ---"
echo "Creando usuario (POST /usuarios)..."
curl -i -X POST "$USUARIOS_URL" \
  -H "Content-Type: application/json" \
  -d '{"idusuario": 10, "nombre": "Carlos", "apellido": "Gómez", "curso": true}'
echo -e "\n------------------------------------------\n"

echo "[2/15] Listando usuarios (GET /usuarios)..."
curl -i -X GET "$USUARIOS_URL"
echo -e "\n------------------------------------------\n"

echo "[3/15] Obteniendo usuario por ID (GET /usuarios/10)..."
curl -i -X GET "$USUARIOS_URL/10"
echo -e "\n------------------------------------------\n"

echo "[4/15] Actualizando usuario (PUT /usuarios/10)..."
curl -i -X PUT "$USUARIOS_URL/10" \
  -H "Content-Type: application/json" \
  -d '{"nombre": "Carlos Alberto", "apellido": "Gómez", "curso": false}'
echo -e "\n------------------------------------------\n"


# SECCIÓN 2: PRUEBAS DE MATERIAS (/materias)


echo "[5/15] Creando materia (POST /materias)..."
curl -i -X POST "$MATERIAS_URL" \
  -H "Content-Type: application/json" \
  -d '{"idmateria": 101, "nombre": "Programación Web", "cuatrimestre": 1, "descripcion": "Materia de Go y Docker"}'
echo -e "\n------------------------------------------\n"

echo "[6/15] Listando materias (GET /materias)..."
curl -i -X GET "$MATERIAS_URL"
echo -e "\n------------------------------------------\n"

echo "[7/15] Obteniendo materia por ID (GET /materias/101)..."
curl -i -X GET "$MATERIAS_URL/101"
echo -e "\n------------------------------------------\n"

echo "[8/15] Actualizando materia (PUT /materias/101)..."
curl -i -X PUT "$MATERIAS_URL/101" \
  -H "Content-Type: application/json" \
  -d '{"nombre": "Programación Web II", "cuatrimestre": 2, "descripcion": "Materia de Go, Docker y Kubernetes"}'
echo -e "\n------------------------------------------\n"


# SECCIÓN 3: PRUEBAS DE VOTOS (/votos)
# Nota: Se omite GET /votos ya que el handler de listar votos fue removido.


echo "[9/15] Creando voto (POST /votos)..."
curl -i -X POST "$VOTOS_URL" \
  -H "Content-Type: application/json" \
  -d '{"idusuario": 10, "idmateria": 101, "puntuacion_dif": 3, "puntuacion_rel": 4, "puntuacion_gusto": 5, "descripcion": "Excelente materia"}'
echo -e "\n------------------------------------------\n"

echo "[10/15] Obteniendo voto por ID compuesto (GET /votos/10/101)..."
curl -i -X GET "$VOTOS_URL/10/101"
echo -e "\n------------------------------------------\n"

echo "[11/15] Actualizando voto (PUT /votos/10/101)..."
curl -i -X PUT "$VOTOS_URL/10/101" \
  -H "Content-Type: application/json" \
  -d '{"puntuacion_dif": 2, "puntuacion_rel": 3, "puntuacion_gusto": 4, "descripcion": "Opinión actualizada de la materia"}'
echo -e "\n------------------------------------------\n"

echo "[12/15] Eliminando voto (DELETE /votos/10/101)..."
curl -i -X DELETE "$VOTOS_URL/10/101"
echo -e "\n------------------------------------------\n"

echo "[13/15] Verificando eliminación de voto (GET /votos/10/101 - Esperado: 404 Not Found)..."
curl -i -X GET "$VOTOS_URL/10/101"
echo -e "\n------------------------------------------\n"

# SECCIÓN 4: ELIMINACIÓN DE MATERIA Y USUARIO Y VERIFICACIÓN 404


echo "[14/15] Eliminando materia (DELETE /materias/101)..."
curl -i -X DELETE "$MATERIAS_URL/101"
echo -e "\n------------------------------------------\n"

echo "[15/15] Eliminando usuario (DELETE /usuarios/10)..."
curl -i -X DELETE "$USUARIOS_URL/10"
echo -e "\n------------------------------------------\n"

echo "Pruebas HTTP completadas con éxito para Usuarios, Materias y Votos."
