# syntax=docker/dockerfile:1.4
FROM golang:1.21-alpine

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o quickdocs ./cmd/server

EXPOSE 8080

CMD ["./quickdocs"]
