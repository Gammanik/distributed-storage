# cmd/storage-node/Dockerfile
FROM golang:1.22.0-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o /storage-node ./cmd/storage-node

FROM alpine:3.18
RUN apk --no-cache add ca-certificates
WORKDIR /app

COPY --from=builder /storage-node /app/storage-node

# Создаем директорию для хранения данных
RUN mkdir -p /data

EXPOSE 9000
ENTRYPOINT ["/app/storage-node"]