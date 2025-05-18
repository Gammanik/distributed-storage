FROM golang:1.22-alpine

WORKDIR /app
COPY . .

RUN cd cmd/rest-server && go build -o /rest-server

EXPOSE 8080
CMD ["/rest-server"]
