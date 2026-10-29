.PHONY: all gen sqlc-gen templ-gen build up down clean restart logs run-local wait test test-full test-unit help

# URL base de la API (se puede sobreescribir: make test BASE_URL=http://localhost:9090)
BASE_URL ?= http://localhost:8080

# Meta por defecto: genera código y levanta la app en Docker
all: gen up

# Generación de código
gen: sqlc-gen templ-gen ## Genera código Go (sqlc) y componentes HTML (templ)

sqlc-gen: ## Genera código Go de base de datos con sqlc
	@echo "Generando código con sqlc..."
	@sqlc generate || echo "sqlc no instalado localmente, usando paquete ./db"

templ-gen: ## Genera componentes HTML con templ
	@echo "Generando componentes con templ..."
	@templ generate || echo "templ no instalado localmente, usando archivos generados"

# Docker e Infraestructura
build: ## Reconstruye las imágenes Docker
	@echo "Construyendo imágenes Docker..."
	docker compose build

up: gen ## Levanta PostgreSQL y la API en segundo plano
	@echo "Levantando contenedores..."
	docker compose up -d --build

down: ## Detiene contenedores y borra volúmenes
	@echo "Deteniendo contenedores..."
	docker compose down -v

clean: down ## Alias para detención y limpieza total

restart: down up ## Reinicia todos los servicios

logs: ## Muestra los logs en tiempo real
	docker compose logs -f

# Ejecución local
run-local: gen ## Ejecuta PostgreSQL en Docker y la API localmente
	@echo "Levantando PostgreSQL..."
	docker compose up -d postgres-db
	@sleep 3
	@echo "Ejecutando main.go localmente..."
	go run main.go

# Pruebas
wait: ## Espera a que la API responda (hasta 60 s)
	@echo "Esperando a que la API responda en $(BASE_URL)..."
	@for i in $$(seq 1 60); do \
		if curl -s -o /dev/null $(BASE_URL)/usuarios; then echo "API en línea."; exit 0; fi; \
		sleep 1; \
	done; \
	echo "Error: la API no respondió a tiempo."; docker compose logs; exit 1

test: ## Ejecuta pruebas de integración HTTP con Hurl (la API ya debe estar levantada)
	@echo "Ejecutando pruebas con Hurl..."
	hurl --test --variable base_url=$(BASE_URL) requests.hurl

test-full: ## Limpia, levanta todo, espera, corre Hurl y limpia al terminar
	@$(MAKE) down
	@$(MAKE) up
	@$(MAKE) wait && $(MAKE) test; rc=$$?; $(MAKE) down; exit $$rc

test-unit: ## Ejecuta pruebas unitarias de Go
	@echo "Ejecutando pruebas unitarias..."
	go test -v ./...

# Ayuda
help: ## Muestra comandos disponibles
	@echo "Comandos del Makefile:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'
