BASE_URL ?= http://localhost:8080

.PHONY: up test down

up: ## Genera código, levanta PostgreSQL en Docker y corre la API local
	sqlc generate
	templ generate
	docker compose up -d --wait database
	go run .

test: ## Espera a la app y corre las pruebas con Hurl
	@timeout 60 sh -c 'until curl -sf -o /dev/null $(BASE_URL)/usuarios; do sleep 1; done'
	hurl --test --variable base_url=$(BASE_URL) requests.hurl

down: ## Detiene PostgreSQL y borra la BD (volumen)
	docker compose down -v
