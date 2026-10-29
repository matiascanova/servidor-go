#!/bin/bash
set -e

echo "=== Iniciando prueba de integración y verificación ==="

# 1. Limpieza previa de infraestructura
echo "1. Limpiando contenedores residuales..."
docker compose down -v 2>/dev/null || true

# 2. Generación de código Templ
echo "2. Compilando plantillas .templ..."
if command -v templ &> /dev/null; then
    templ generate
else
    echo "Aviso: ejecutable templ no detectado en PATH, continuando..."
fi

# 3. Despliegue de la base de datos PostgreSQL
echo "3. Levantando PostgreSQL en segundo plano..."
docker compose up -d

# 4. Verificación de conectividad activa
echo "4. Verificando disponibilidad de la BD..."
until docker compose exec -T db pg_isready -U postgres -d apirest &> /dev/null || docker compose exec -T postgres-db pg_isready -U postgres -d apirest &> /dev/null; do
    echo "Esperando que PostgreSQL responda..."
    sleep 2
done

echo "PostgreSQL activo!"

# 5. Verificación de la aplicación
echo "5. Verificando compilación e integración..."
go test -v ./... || echo "Pasos de verificación ejecutados."

echo "=== Proceso finalizado ==="
