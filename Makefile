.PHONY: all gen sqlc-gen templ-gen build up down clean restart logs run-local test test-unit help

# Meta por defecto: Genera código (sqlc + templ) y levanta la infraestructura en Docker
all: gen up

# 1. GENERACIÓN DE CÓDIGO
gen: sqlc-gen templ-gen ## Genera el código Go desde SQL (sqlc) y los componentes HTML (templ)

sqlc-gen: ## Genera código Go de persistencia a partir de queries.sql y schema.sql
	@echo "Generando código de persistencia con sqlc..."
	@sqlc generate || echo "Advertencia: 'sqlc' no está instalado localmente, utilizando paquete ./db existente."

templ-gen: ## Genera archivos Go a partir de las plantillas .templ
	@echo "Generando componentes HTML con templ..."
	@templ generate || echo "Advertencia: 'templ' no está instalado localmente, utilizando componentes generados existentes."

# 2. DOCKER & INFRAESTRUCTURA
build: ## Reconstruye las imágenes Docker de la aplicación
	@echo "Construyendo imágenes Docker..."
	docker compose build

up: gen ## Levanta la infraestructura completa en segundo plano (PostgreSQL + API Go)
	@echo "Levantando contenedores en Docker Compose..."
	docker compose up -d --build

down: ## Detiene los contenedores y elimina volúmenes/redes
	@echo "Deteniendo contenedores y limpiando volúmenes..."
	docker compose down -v

clean: down ## Alias de 'down' para limpieza completa del entorno

restart: down up ## Reinicio limpio de todos los servicios

logs: ## Muestra los logs en tiempo real de todos los contenedores
	docker compose logs -f

# 3. EJECUCIÓN LOCAL
run-local: gen ## Levanta solo PostgreSQL en Docker y ejecuta la app Go localmente
	@echo "Levantando PostgreSQL en Docker..."
	docker compose up -d postgres-db
	@echo "Esperando a que PostgreSQL esté listo..."
	@sleep 3
	@echo "Ejecutando main.go localmente..."
	go run main.go

# 4. PRUEBAS & SUITE DE VALIDACIÓN
test: ## Ejecuta la suite de pruebas de integración HTTP end-to-end
	@echo "Ejecutando suite de pruebas HTTP de integración..."
	@chmod +x requests-v11.sh 2>/dev/null || chmod +x requests.sh 2>/dev/null
	@if [ -f requests-v11.sh ]; then ./requests-v11.sh; else ./requests.sh; fi

test-unit: ## Ejecuta las pruebas unitarias Table-Driven de Go
	@echo "Ejecutando pruebas unitarias en Go..."
	go test -v ./...

# 5. AYUDA
help: ## Muestra este menú de ayuda
	@echo "Comandos disponibles en el Makefile:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'
