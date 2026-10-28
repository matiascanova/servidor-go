# Dockerfile: Compilación multi-etapa para Go + Templ
FROM golang:1.24-alpine AS builder

WORKDIR /app

# 1. Instalar git (necesario para que 'go install' descargue de GitHub en Alpine)
RUN apk add --no-cache git

# 2. Instalar el CLI de Templ
RUN go install github.com/a-h/templ/cmd/templ@latest

COPY go.mod go.sum ./
RUN go mod download || true

COPY . .

# 3. Generar archivos Go desde las plantillas .templ
RUN templ generate

# 4. Compilar el ejecutable estático
RUN CGO_ENABLED=0 GOOS=linux go build -o server main.go

# Etapa 2: Imagen final liviana para ejecución
FROM alpine:3.20

WORKDIR /app

COPY --from=builder /app/server .
COPY --from=builder /app/static ./static

EXPOSE 8080

CMD ["./server"]
