# exchange-grpc

Учебный проект биржевой системы на gRPC — монорепо из 4 микросервисов.

## Структура

```
proto/                  — .proto контракты + generated pb/
shared/                 — logger, JWT, gRPC interceptors
userservice/            — Register, Login, JWT (:50050)
spotservice/            — ViewMarkets, GetMarket (:50051)
orderservice/           — CreateOrder, StreamOrderUpdates (:50052)
orderserviceclient/     — CLI-клиент
infrastructure/compose/ — docker-compose
test/integration/       — интеграционные тесты
```

## Требования

- Go 1.22+
- [protoc](https://grpc.io/docs/protoc-installation/) или [Buf](https://buf.build/docs/installation/)

## Быстрый старт

```bash
make proto
make build-services

# в отдельных терминалах:
make run-user
make run-spot-service
make run-order-service

# создать заказ:
make run-client
```

Docker:

```bash
make compose-up
```

## Конфигурация

| Переменная | По умолчанию | Сервис | Описание |
|------------|--------------|--------|----------|
| `USER_SERVICE_ADDR` | `:50050` | userservice | gRPC listen |
| `USER_SERVICE_HOST` | `localhost:50050` | client | Адрес userservice |
| `SPOT_SERVICE_ADDR` | `:50051` | spotservice | gRPC listen |
| `SPOT_SERVICE_HOST` | `localhost:50051` | orderservice, client | Адрес spotservice |
| `ORDER_SERVICE_ADDR` | `:50052` | orderservice | gRPC listen |
| `ORDER_SERVICE_HOST` | `localhost:50052` | client | Адрес orderservice |
| `JWT_SECRET` | `dev-exchange-secret` | все | Общий секрет подписи JWT |
| `JWT_TTL` | `24h` | userservice | Время жизни access token |
| `SPOT_GRPC_TIMEOUT` | `5s` | orderservice | Таймаут вызовов к spot |
| `ORDER_HUB_BUFFER_SIZE` | `256` | orderservice | Буфер hub обновлений |

## Безопасность (JWT)

`user_id` и роли **не в теле запроса** — только через Bearer JWT (`Authorization` metadata). Токен выпускает `userservice` при Login. Order → Spot пробрасывает JWT автоматически.

## RBAC

Роли в JWT claims (`roles`). Рынки с пустым `allowed_roles` доступны всем. Seed `BNB-USDT` требует `trader` или `admin`.

## CLI-клиент

```bash
go run ./orderserviceclient \
  --register \
  --email=user@example.com \
  --password=password123 \
  --market-id=BTC-USDT \
  --order-side=buy \
  --price=42000 \
  --quantity=0.01
```

Флаги: `--email`, `--password`, `--register`, `--market-id`, `--order-side` (`buy`|`sell`), `--price`, `--quantity`, `--addr`, `--user-addr`, `--request-id`.

## API

Proto-контракты: [`proto/proto/`](proto/proto/)

| Сервис | RPC | Описание |
|--------|-----|----------|
| UserService | `Register` | Регистрация |
| UserService | `Login` | Вход, выпуск JWT |
| SpotService | `ViewMarkets` | Список активных рынков |
| SpotService | `GetMarket` | Рынок по ID |
| OrderService | `CreateOrder` | Создание заказа |
| OrderService | `GetOrderStatus` | Статус заказа |
| OrderService | `StreamOrderUpdates` | Server-stream обновлений |

## Тесты

```bash
make test
make test-integration
```
