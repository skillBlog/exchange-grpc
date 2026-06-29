# syntax=docker/dockerfile:1

FROM golang:1.23-alpine AS builder

RUN apk add --no-cache ca-certificates git

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG SERVICE
RUN test -n "$SERVICE"
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/service "./cmd/${SERVICE}"

FROM alpine:3.20

RUN apk add --no-cache ca-certificates wget

WORKDIR /app

COPY --from=builder /out/service /app/service

USER nobody

ENTRYPOINT ["/app/service"]
