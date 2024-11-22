FROM golang:1.23 AS builder

WORKDIR /app

ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o manager manager.go mongo.go
RUN go build -o storage storage.go

FROM alpine:latest

RUN apk add --no-cache bash

WORKDIR /app

COPY --from=builder /app/manager /app/manager
COPY --from=builder /app/storage /app/storage

RUN chmod +x /app/manager /app/storage

EXPOSE 19000-19010
EXPOSE 18080