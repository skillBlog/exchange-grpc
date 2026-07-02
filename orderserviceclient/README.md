# orderserviceclient

CLI для тестирования gRPC API биржи.

## Запуск

```bash
make run-client
# или
go run . --register --email=user@example.com --password=password123 \
  --market-id=BTC-USDT --order-side=buy --quantity=0.01
```

## Конфигурация

| Переменная | По умолчанию |
|------------|--------------|
| `USER_SERVICE_HOST` | `localhost:50050` |
| `SPOT_SERVICE_HOST` | `localhost:50051` |
| `ORDER_SERVICE_HOST` | `localhost:50052` |
