# orderservice

Сервис управления ордерами.

## Запуск

```bash
go run .
make run-order-service
```

| Переменная | По умолчанию | Описание |
|------------|--------------|----------|
| `ORDER_SERVICE_ADDR` | `:50052` | gRPC-адрес |
| `SPOT_SERVICE_HOST` | `localhost:50051` | Адрес spotservice |
