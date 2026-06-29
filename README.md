# exchange-grpc

Учебный проект биржевой системы на gRPC: **OrderService** и **SpotInstrumentService**.

## Требования

- Go 1.22+
- [protoc](https://grpc.io/docs/protoc-installation/) или [Buf](https://buf.build/docs/installation/)

## Быстрый старт

```bash
# Установить плагины и сгенерировать Go-код из .proto
make proto
# Windows PowerShell:
# .\scripts\generate-proto.ps1

# Сборка
make build
```

## Конфигурация

Переменные окружения (значения по умолчанию):

| Переменная | По умолчанию | Описание |
|------------|--------------|----------|
| `SPOT_INSTRUMENT_ADDR` | `:50051` | Адрес SpotInstrumentService |
| `ORDER_ADDR` | `:50052` | Адрес OrderService |
| `ORDER_HOST` | `localhost:50052` | Адрес OrderService для CLI-клиента |
| `SPOT_METRICS_ADDR` | `:9090` | Prometheus metrics SpotInstrumentService |
| `ORDER_METRICS_ADDR` | `:9091` | Prometheus metrics OrderService |
| `SPOT_INSTRUMENT_HOST` | `localhost:50051` | Адрес SpotInstrument для OrderService client |
| `REDIS_ADDR` | _(пусто)_ | Redis для кэша ViewMarkets (например `localhost:6379`) |
| `REDIS_CACHE_TTL` | `30s` | TTL кэша ViewMarkets |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | _(пусто)_ | OTLP endpoint Jaeger (`http://localhost:4318`) |

## Docker Compose (полный стек)

Поднимает **Redis**, SpotInstrumentService, OrderService и Prometheus в одной сети.

```bash
make compose-up
make compose-logs   # опционально
```

| Сервис | gRPC | Metrics | Описание |
|--------|------|---------|----------|
| spot-instrument | localhost:50051 | localhost:9090/metrics | Рынки (+ Redis cache) |
| redis | localhost:6379 | — | Кэш ViewMarkets |
| jaeger | — | http://localhost:16686 | UI трассировки |
| order-service | localhost:50052 | localhost:9091/metrics | Заказы |
| prometheus | — | http://localhost:9092 | UI и scrape |

Проверка после старта:

```bash
go run ./cmd/client \
  --addr=localhost:50052 \
  --user-id=user-1 \
  --market-id=BTC-USDT \
  --order-type=limit \
  --price=42000 \
  --quantity=0.01
```

Остановка:

```bash
make compose-down
```

## Jaeger tracing

При заданном `OTEL_EXPORTER_OTLP_ENDPOINT` сервисы экспортируют trace через OTLP.
`x-request-id` добавляется в span как атрибут `request_id`, а W3C trace context
пробрасывается между OrderService и SpotInstrumentService.

```bash
# Docker Compose поднимает Jaeger автоматически
make compose-up

# Локально
docker run -d --name jaeger -p 16686:16686 -p 4318:4318 -e COLLECTOR_OTLP_ENABLED=true jaegertracing/all-in-one:1.62.0
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318 make run-spot
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318 make run-order
```

UI: http://localhost:16686 — ищите trace `order-service` → `spot-instrument-service`.

## RBAC (user_roles)

Рынки с пустым `allowed_roles` доступны всем. Ограниченные рынки (seed: `BNB-USDT` → `trader`, `admin`) видны и доступны для торговли только при совпадении роли.

| RPC | Поле | Поведение |
|-----|------|-----------|
| `ViewMarkets` | `user_roles` | Возвращает только активные рынки, доступные пользователю |
| `CreateOrder` | `user_roles` | `PermissionDenied`, если роль не подходит для `market_id` |

```bash
# BNB-USDT требует роль trader или admin
go run ./cmd/client \
  --user-id=user-1 \
  --market-id=BNB-USDT \
  --order-type=market \
  --quantity=1 \
  --user-roles=trader
```

## Prometheus (только контейнер)

Если сервисы запущены локально на хосте, можно поднять только Prometheus — для этого в `deployments/prometheus/prometheus.yml` замените targets на `host.docker.internal:9090` и `:9091`.

```bash
cd deployments
docker compose up -d prometheus
```

UI Prometheus: http://localhost:9092

## CLI-клиент

```bash
# Сначала запустите spot-instrument-service и order-service
go run ./cmd/client \
  --user-id=user-1 \
  --market-id=BTC-USDT \
  --order-type=limit \
  --price=42000 \
  --quantity=0.01
```

Флаги: `--user-id`, `--market-id`, `--order-type` (`limit`|`market`), `--price`, `--quantity`, `--addr`, `--request-id`, `--user-roles` (через запятую).

## API

Proto-контракт: [`api/proto/exchange/v1/exchange.proto`](api/proto/exchange/v1/exchange.proto)

| Сервис | RPC | Описание |
|--------|-----|----------|
| SpotInstrumentService | `ViewMarkets` | Список активных рынков |
| SpotInstrumentService | `GetMarket` | Рынок по ID (в т.ч. неактивный) |
| OrderService | `CreateOrder` | Создание заказа |
| OrderService | `GetOrderStatus` | Статус заказа |
| OrderService | `StreamOrderUpdates` | Server-stream обновлений статуса |

## Структура проекта

```
api/proto/          — protobuf контракты и generated code
cmd/                — точки входа (сервисы, клиент)
internal/domain/    — доменные сущности
internal/usecase/   — бизнес-логика
internal/adapter/   — gRPC, репозитории, клиенты
internal/platform/  — config, logger, interceptors, metrics
deployments/        — Docker, Prometheus
test/integration/   — интеграционные тесты
```

```bash
make test-integration
```
