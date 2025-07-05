# cmd/rest-server/Dockerfile
FROM golang:1.22.0-alpine AS builder

# Установка зависимостей для CGO
RUN apk add --no-cache gcc musl-dev

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
# BoltDB требует CGO для работы
RUN CGO_ENABLED=1 go build -o /rest-server ./cmd/rest-server

FROM alpine:3.18
# Устанавливаем runtime зависимости
RUN apk --no-cache add ca-certificates libc6-compat

WORKDIR /app

COPY --from=builder /rest-server /app/rest-server

# Создаем директорию для хранения метаданных
RUN mkdir -p /data

EXPOSE 8080
ENTRYPOINT ["/app/rest-server"]