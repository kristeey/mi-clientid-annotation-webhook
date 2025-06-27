# syntax=docker/dockerfile:1
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY main.go ./
RUN go build -o init-container main.go

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/init-container ./init-container
ENTRYPOINT ["/app/init-container"]
