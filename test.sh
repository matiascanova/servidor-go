#!/bin/sh

echo "limpieza inicial"
docker compose down -v

echo "llamada a sqlc"
sqlc generate

echo "levantar docker"
docker compose up -d

echo "sleep hasta que la base se levante correctamente"
while ! docker exec postgres-db pg_isready -U postgres > /dev/null 2>&1; do
    echo "base todavia no lista"
    sleep 1
done

echo "ejecucion testeo"
go test -v ./...

echo "limpieza final"
docker compose down -v

echo "FIN DE TESTEO"
