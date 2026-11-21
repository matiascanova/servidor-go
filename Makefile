BASE_URL ?= http://localhost:8080
COMPOSE  ?= docker compose

.PHONY: up test down

up:
	sqlc generate
	templ generate
	-@$(COMPOSE) down -v 
	@trap 'echo "Limpiando contenedores..."; $(COMPOSE) down -v ' EXIT; \
	trap 'exit 130' INT TERM; \
	$(COMPOSE) up -d --wait database && go run .

# run_id hace únicos los emails de los usuarios que crean las pruebas en cada corrida
test:
	@timeout 60 sh -c 'until curl -sf -o /dev/null $(BASE_URL)/; do sleep 1; done'
	hurl --test --variable base_url=$(BASE_URL) --variable run_id=$$(date +%s)-$$$$ requests.hurl

down:
	$(COMPOSE) down -v --remove-orphans
