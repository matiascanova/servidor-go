#!/usr/bin/env bash

set -e

BASE_URL="http://localhost:8080"
USUARIOS_URL="$BASE_URL/usuarios"
MATERIAS_URL="$BASE_URL/materias"
VOTOS_URL="$BASE_URL/votos"

# 0. Verificación simple del servicio de Docker (sin intento de inicio automático)
echo "Verificando el estado del servicio Docker..."
if ! docker info > /dev/null 2>&1; then
  echo "Error: El demonio de Docker no está en ejecución."
  echo "Por favor, iniciá Docker Desktop / servicio de Docker y volvé a ejecutar el script."
  exit 1
fi
echo "Servicio de Docker activo y respondiendo."

# Función de limpieza al salir
cleanup() {
  echo " Limpiando contenedores y volúmenes Docker..."
  docker compose down -v 2>/dev/null || true
}
trap cleanup EXIT

echo " 1. Limpiando ejecuciones previas y volúmenes"
docker compose down -v 2>/dev/null || true

echo " 2. Levantando infraestructura con Docker Compose"
docker compose up -d --build

echo -e "\nEsperando a que la base de datos PostgreSQL esté completamente lista..."
MAX_PG_RETRIES=30
COUNT=0
until docker exec postgres-db pg_isready -U postgres -d apirest > /dev/null 2>&1 || [ $COUNT -eq $MAX_PG_RETRIES ]; do
  sleep 1
  COUNT=$((COUNT + 1))
  echo -n "."
done
echo ""

if [ $COUNT -eq $MAX_PG_RETRIES ]; then
  echo "Error: PostgreSQL no estuvo listo a tiempo."
  docker compose logs postgres-db
  exit 1
fi
echo "Base de datos PostgreSQL lista y aceptando conexiones."

echo -e "\nEsperando a que la API REST responda en http://localhost:8080/..."
MAX_RETRIES=30
COUNT=0
until curl -s "$USUARIOS_URL" > /dev/null 2>&1 || [ $COUNT -eq $MAX_RETRIES ]; do
  sleep 1
  COUNT=$((COUNT + 1))
  echo -n "."
done
echo ""

if [ $COUNT -eq $MAX_RETRIES ]; then
  echo "Error: La API REST no respondió a tiempo."
  echo "Mostrando logs de los contenedores:"
  docker compose logs
  exit 1
fi

echo "API REST en línea. Iniciando suite completa de pruebas HTTP..."
echo -e "\n==========================================\n"

# FUNCIÓN DE ASERCIÓN Y PRUEBA DE ENDPOINTS HTTP
test_endpoint() {
  local paso="$1"
  local descripcion="$2"
  local status_esperado="$3"
  local contenido_esperado="$4"
  local metodo="$5"
  local url="$6"
  local body="$7"

  echo "[$paso/15] $descripcion ($metodo $url)..."

  local tmp_res
  tmp_res=$(mktemp)

  local http_code
  if [ -n "$body" ]; then
    http_code=$(curl -s -o "$tmp_res" -w "%{http_code}" -X "$metodo" "$url" \
      -H "Content-Type: application/json" \
      -d "$body")
  else
    http_code=$(curl -s -o "$tmp_res" -w "%{http_code}" -X "$metodo" "$url")
  fi

  local res_body
  res_body=$(cat "$tmp_res")
  rm -f "$tmp_res"

  echo "   -> Código HTTP obtenido: $http_code (Esperado: $status_esperado)"
  if [ -n "$res_body" ]; then
    echo "   -> Respuesta: $res_body"
  fi

  # Verificación del Código HTTP
  if [ "$http_code" -ne "$status_esperado" ]; then
    echo "ERROR en Paso $paso: Se esperaba el código HTTP $status_esperado pero se obtuvo $http_code"
    echo "Logs del contenedor Go API:"
    docker compose logs go-api
    exit 1
  fi

  # Verificación del contenido esperado (si se especificó)
  if [ -n "$contenido_esperado" ]; then
    if ! echo "$res_body" | grep -q "$contenido_esperado"; then
      echo "ERROR en Paso $paso: El contenido retornado no incluye '$contenido_esperado'"
      exit 1
    fi
  fi

  echo "   Paso $paso exitoso."
  echo -e "------------------------------------------\n"
}

# SECCIÓN 1: PRUEBAS DE USUARIOS (/usuarios)
echo "--- SECCIÓN 1: PRUEBAS DE USUARIOS ---"
test_endpoint "1" "Crear usuario" 201 "Carlos" "POST" "$USUARIOS_URL" '{"idusuario": 10, "nombre": "Carlos", "apellido": "Gómez", "curso": true}'
test_endpoint "2" "Listar usuarios" 200 "Carlos" "GET" "$USUARIOS_URL" ""
test_endpoint "3" "Obtener usuario por ID" 200 "Carlos" "GET" "$USUARIOS_URL/10" ""
test_endpoint "4" "Actualizar usuario" 200 "Carlos Alberto" "PUT" "$USUARIOS_URL/10" '{"nombre": "Carlos Alberto", "apellido": "Gómez", "curso": false}'

# SECCIÓN 2: PRUEBAS DE MATERIAS (/materias)
echo "--- SECCIÓN 2: PRUEBAS DE MATERIAS ---"
test_endpoint "5" "Crear materia" 201 "Programación Web" "POST" "$MATERIAS_URL" '{"idmateria": 101, "nombre": "Programación Web", "cuatrimestre": 1, "descripcion": "Materia de Go y Docker"}'
test_endpoint "6" "Listar materias" 200 "Programación Web" "GET" "$MATERIAS_URL" ""
test_endpoint "7" "Obtener materia por ID" 200 "Programación Web" "GET" "$MATERIAS_URL/101" ""
test_endpoint "8" "Actualizar materia" 200 "Programación Web II" "PUT" "$MATERIAS_URL/101" '{"nombre": "Programación Web II", "cuatrimestre": 2, "descripcion": "Materia de Go, Docker y Kubernetes"}'

# SECCIÓN 3: PRUEBAS DE VOTOS (/votos)
echo "--- SECCIÓN 3: PRUEBAS DE VOTOS (Relaciona Usuario 10 y Materia 101) ---"
test_endpoint "9" "Crear voto" 201 "exitosa" "POST" "$VOTOS_URL" '{"idusuario": 10, "idmateria": 101, "puntuacion_dif": 3, "puntuacion_rel": 4, "puntuacion_gusto": 5, "descripcion": "Excelente materia"}'
test_endpoint "10" "Obtener voto por ID compuesto" 200 "Excelente materia" "GET" "$VOTOS_URL/10/101" ""
test_endpoint "11" "Actualizar voto" 200 "Opinión actualizada" "PUT" "$VOTOS_URL/10/101" '{"puntuacion_dif": 2, "puntuacion_rel": 3, "puntuacion_gusto": 4, "descripcion": "Opinión actualizada de la materia"}'
test_endpoint "12" "Eliminar voto" 204 "" "DELETE" "$VOTOS_URL/10/101" ""
test_endpoint "13" "Verificar eliminación de voto (404)" 404 "no encontrado" "GET" "$VOTOS_URL/10/101" ""

# SECCIÓN 4: ELIMINACIÓN Y VERIFICACIÓN DE INTEGRIDAD
echo "--- SECCIÓN 4: ELIMINACIÓN Y LIMPIEZA ---"
test_endpoint "14" "Eliminar materia" 204 "" "DELETE" "$MATERIAS_URL/101" ""
test_endpoint "15" "Eliminar usuario" 204 "" "DELETE" "$USUARIOS_URL/10" ""

echo " Pruebas HTTP completadas con éxito para Usuarios, Materias y Votos."
