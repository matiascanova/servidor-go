BASE_URL ?= http://localhost:8080
COMPOSE  ?= docker compose

.PHONY: up test down

up:
	sqlc generate
	templ generate
	-@$(COMPOSE) down -v --remove-orphans
	@trap 'echo "Limpiando contenedores..."; $(COMPOSE) down -v --remove-orphans' EXIT; \
	trap 'exit 130' INT TERM; \
	$(COMPOSE) up -d --wait database && go run .

test:
	@timeout 60 sh -c 'until curl -sf -o /dev/null $(BASE_URL)/; do sleep 1; done'
	hurl --test --variable base_url=$(BASE_URL) requests.hurl

down:
	$(COMPOSE) down -v --remove-orphans
