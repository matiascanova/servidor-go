.PHONY: all build run generate clean docker-up docker-down test

all: generate build

# Generar código Go desde plantillas Templ
generate:
	@echo "Generando código Go desde plantillas .templ..."
	templ generate

# Compilar la aplicación Go
build: generate
	@echo "Compilando binario ejecutable server..."
	go build -o server main.go

# Ejecutar el servidor localmente
run: generate
	@echo "Iniciando servidor HTTP..."
	go run main.go

# Levantar base de datos e infraestructura Docker
docker-up:
	docker compose up -d

# Detener contenedores Docker y remover volúmenes
docker-down:
	docker compose down -v

# Limpiar binarios y archivos generados
clean:
	rm -f server
	rm -rf views/*_templ.go
