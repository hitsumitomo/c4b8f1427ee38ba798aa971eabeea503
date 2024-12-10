FROM golang:1.23 AS builder

WORKDIR /app

ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o /app/manager cmd/manager/manager.go
RUN go build -o /app/storage cmd/storage/storage.go

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/manager /app/manager
COPY --from=builder /app/storage /app/storage

EXPOSE 19000-19010
EXPOSE 18080
