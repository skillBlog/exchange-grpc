# exchange-grpc

Учебный проект биржевой системы на gRPC — монорепо из 4 микросервисов с PostgreSQL, Redis и JWT-аутентификацией.

## Структура

```
proto/                  — .proto контракты + generated pb/
shared/                 — logger, JWT, gRPC interceptors, errors, redis, health
userservice/            — Register, Login, JWT, refresh tokens (:50050)
spotservice/            — ViewMarkets, GetMarket (:50051)
orderservice/           — CreateOrder, ListOrders, StreamOrderUpdates (:50052)
orderserviceclient/     — CLI-клиент
infrastructure/compose/ — docker-compose (postgres, redis, 3 сервиса)
test/integration/       — интеграционные тесты (bufconn, in-process)
```

## Требования

- Go 1.25+
- [protoc](https://grpc.io/docs/protoc-installation/) или [Buf](https://buf.build/docs/installation/) — для `make proto`
- Docker — для `make compose-up` (PostgreSQL + Redis)

## Быстрый старт

### Локально (in-memory в интеграционных тестах; сервисы — с Postgres)

```bash
make proto          # перегенерация pb/ (нужен buf или bash)
make build-services
make test
make test-integration
```

Запуск сервисов по отдельности (нужен Postgres и Redis для orderservice):

```bash
make run-user
make run-spot-service
make run-order-service

# создать заказ через CLI:
make run-client
```

### Docker (рекомендуется)

Поднимает PostgreSQL, Redis и все три сервиса с миграциями:

```bash
make compose-up
make compose-down   # остановить
```

Порты: userservice `:50050`, spotservice `:50051`, orderservice `:50052`, postgres `:5432`, redis `:6379`.

## Конфигурация

Общие переменные:

| Переменная | По умолчанию | Описание |
|------------|--------------|----------|
| `JWT_SECRET` | `dev-exchange-secret` | Секрет подписи JWT (общий для всех сервисов) |
| `JWT_ACCESS_TTL` | `15m` | Время жизни access token (fallback: `JWT_TTL`) |
| `JWT_REFRESH_TTL` | `168h` | Время жизни refresh token (userservice) |

### userservice

| Переменная | По умолчанию | Описание |
|------------|--------------|----------|
| `USER_SERVICE_ADDR` | `:50050` | gRPC listen |
| `USER_DATABASE_URL` | `postgres://exchange:exchange@localhost:5432/userservice?sslmode=disable` | PostgreSQL |
| `USER_MIGRATIONS_DIR` | `migrations` | Путь к goose-миграциям |
| `LOGIN_RATE_LIMIT` | `5` | Лимит попыток Login на email в окне |
| `LOGIN_RATE_WINDOW` | `1m` | Окно rate limit для Login |

### spotservice

| Переменная | По умолчанию | Описание |
|------------|--------------|----------|
| `SPOT_SERVICE_ADDR` | `:50051` | gRPC listen |
| `SPOT_SERVICE_HOST` | `localhost:50051` | Адрес для клиентов |
| `SPOT_DATABASE_URL` | `postgres://exchange:exchange@localhost:5432/spotservice?sslmode=disable` | PostgreSQL |
| `SPOT_MIGRATIONS_DIR` | `migrations` | Путь к goose-миграциям |
| `MARKET_CACHE_TTL` | `30s` | TTL in-memory кеша рынков |
| `VIEW_MARKETS_RATE_LIMIT` | `30` | Лимит ViewMarkets на user_id |
| `VIEW_MARKETS_RATE_WINDOW` | `1m` | Окно rate limit |

### orderservice

| Переменная | По умолчанию | Описание |
|------------|--------------|----------|
| `ORDER_SERVICE_ADDR` | `:50052` | gRPC listen |
| `ORDER_SERVICE_HOST` | `localhost:50052` | Адрес для клиентов |
| `SPOT_SERVICE_HOST` | `localhost:50051` | Адрес spotservice |
| `SPOT_GRPC_TIMEOUT` | `5s` | Таймаут вызовов к spot |
| `ORDER_DATABASE_URL` | `postgres://exchange:exchange@localhost:5432/orderservice?sslmode=disable` | PostgreSQL |
| `ORDER_MIGRATIONS_DIR` | `migrations` | Путь к goose-миграциям |
| `ORDER_HUB_BUFFER_SIZE` | `256` | Буфер hub обновлений ордеров |
| `REDIS_URL` | `redis://localhost:6379/0` | Redis для rate limit CreateOrder |
| `CREATE_ORDER_GLOBAL_RATE_LIMIT` | `20000` | Глобальный лимит CreateOrder |
| `CREATE_ORDER_GLOBAL_RATE_WINDOW` | `1m` | Окно глобального лимита |
| `CREATE_ORDER_RATE_LIMIT_USER` | `10` | Per-user лимит (роль `user`) |
| `CREATE_ORDER_RATE_LIMIT_TRADER` | `100` | Per-user лимит (роль `trader`) |
| `CREATE_ORDER_RATE_LIMIT_ADMIN` | `1000` | Per-user лимит (роль `admin`) |
| `CREATE_ORDER_RATE_WINDOW` | `1m` | Окно per-user лимита |

Если Redis недоступен, orderservice использует in-memory rate limiter (с предупреждением в лог).

## Безопасность (JWT)

`user_id` и роли **не в теле запроса** — только через Bearer JWT (`Authorization` metadata). Access token выпускает `userservice` при Login/RefreshToken. Order → Spot пробрасывает JWT автоматически.

При регистрации пользователю назначается роль `user` (не из request).

## RBAC

Роли в JWT claims (`roles`), нормализуются к lower-case. Рынки с пустым `allowed_roles` доступны всем. Seed `BNB-USDT` требует `trader` или `admin`.

Rate limit CreateOrder зависит от роли: `user` < `trader` < `admin`.

## Идемпотентность CreateOrder

`CreateOrderRequest` поддерживает `idempotency_key`. Повторный запрос с тем же ключом и `user_id` возвращает уже созданный ордер (хранится в PostgreSQL).

## CLI-клиент

```bash
go run ./orderserviceclient \
  --register \
  --email=user@example.com \
  --password=password123 \
  --market-id=BTC-USDT \
  --order-side=buy \
  --quantity=0.01
```

Флаги: `--email`, `--password`, `--register`, `--market-id`, `--order-side` (`buy`|`sell`), `--price`, `--quantity`, `--addr`, `--user-addr`, `--request-id`.

## API

Proto-контракты: [`proto/proto/`](proto/proto/)

| Сервис | RPC | Описание |
|--------|-----|----------|
| UserService | `Register` | Регистрация (роль `user` по умолчанию) |
| UserService | `Login` | Вход, выпуск access + refresh token |
| UserService | `RefreshToken` | Обновление access token |
| UserService | `GetUser` | Профиль текущего пользователя |
| UserService | `Logout` | Отзыв refresh token |
| SpotService | `ViewMarkets` | Список активных рынков (cursor pagination) |
| SpotService | `GetMarket` | Рынок по ID |
| OrderService | `CreateOrder` | Создание market-ордера |
| OrderService | `GetOrderStatus` | Статус ордера |
| OrderService | `ListOrders` | Список ордеров пользователя (cursor pagination) |
| OrderService | `StreamOrderUpdates` | Server-stream обновлений статуса |

## Health checks

Все сервисы экспонируют gRPC health (`grpc.health.v1`):

- **userservice** — PostgreSQL ping
- **spotservice** — PostgreSQL + market cache
- **orderservice** — PostgreSQL + Redis (если подключён)

## Graceful shutdown

По SIGINT/SIGTERM сервисы выполняют `GracefulStop` gRPC (таймаут 15s), затем закрывают соединения с БД/Redis.

## Тесты

```bash
make test              # unit + integration
make test-integration  # только integration (-v)
make test-race         # с -race
make lint              # go vet
make check             # build + test + lint
```

Интеграционные тесты поднимают сервисы in-process через `bufconn` — внешний Postgres/Redis для тестов не нужен.
