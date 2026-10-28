.PHONY: all sqlc build up test clean restart

# Meta por defecto: regenera sqlc, levanta contenedores y ejecuta los tests
all: sqlc up test

# 1. Regenera el código Go de persistencia con sqlc
sqlc:
	@echo "Generando código de persistencia con sqlc..."
	@sqlc generate || echo "⚠️ Advertencia: 'sqlc' no está instalado localmente, se utilizará el paquete ./db existente."

# 2. Compila las imágenes de Docker Compose
build:
	docker compose build

# 3. Levanta la infraestructura (PostgreSQL + API Go)
up:
	docker compose up -d --build

# 4. Ejecuta la suite completa de pruebas HTTP
test:
	chmod +x requests.sh
	./requests.sh

# 5. Limpia contenedores, redes y volúmenes
clean:
	docker compose down -v

# 6. Reinicio limpio del entorno
restart: clean up
