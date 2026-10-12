# Dockerfile-v5: Multi-stage build optimizado para API Go en Docker Compose

# Etapa 1: Compilación (Builder)
FROM golang:1.24-alpine AS builder

# Habilitar descarga automática del toolchain Go si go.mod exige go >= 1.25.0
ENV GOTOOLCHAIN=auto

WORKDIR /app

# Copiar archivos de dependencias de Go
COPY go.mod go.sum* ./
RUN go mod download || true

# Copiar todo el código fuente del proyecto
COPY . .

# Compilar un binario estático optimizado para Linux (CGO_ENABLED=0 para portabilidad)
RUN CGO_ENABLED=0 GOOS=linux go build -o main .

# Etapa 2: Imagen final de producción (Ejecución)
FROM alpine:3.20

RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copiar únicamente el ejecutable desde la etapa de compilación
COPY --from=builder /app/main .

EXPOSE 8080

CMD ["./main"]
