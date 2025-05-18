FROM golang:1.22-alpine

WORKDIR /app
COPY . .

RUN cd cmd/storage-node && go build -o /storage-node

EXPOSE 9000
CMD ["/storage-node"]
