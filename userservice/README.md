# userservice

Сервис аутентификации и управления пользователями.

## Запуск

```bash
go run .
# или из корня монорепо:
make run-user
```

Переменные окружения:

| Переменная | По умолчанию | Описание |
|------------|--------------|----------|
| `USER_SERVICE_ADDR` | `:50050` | gRPC-адрес |

## Структура

```
internal/
  application/   — use cases (этап 3)
  domain/        — user domain
  infrastructure/
  interfaces/grpcserver/
```
