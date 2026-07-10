# syntax=docker/dockerfile:1

FROM golang:1.25-alpine AS builder

RUN apk add --no-cache ca-certificates git

WORKDIR /src

COPY go.work ./
COPY proto/ proto/
COPY shared/ shared/
COPY spotservice/ spotservice/

WORKDIR /src/spotservice
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/service .

FROM alpine:3.20

RUN apk add --no-cache ca-certificates

WORKDIR /app
COPY --from=builder /out/service /app/service
COPY --from=builder /src/spotservice/migrations /app/migrations

ENV SPOT_MIGRATIONS_DIR=/app/migrations

USER nobody
ENTRYPOINT ["/app/service"]
