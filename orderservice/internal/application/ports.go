package application

import (
	"context"

	"github.com/exchange-grpc/orderservice/internal/domain"
)

// MarketChecker проверяет, что рынок существует, доступен для торговли и разрешён пользователю.
// При успехе возвращает торговые лимиты рынка (пустые поля = правило не задано).
type MarketChecker interface {
	EnsureMarketAvailable(ctx context.Context, marketID string, userRoles []string) (domain.MarketLimits, error)
}

// CreateOrderRateLimiter ограничивает частоту CreateOrder (глобально и per-user).
type CreateOrderRateLimiter interface {
	Allow(ctx context.Context, userID string, userRoles []string) error
}

// TxManager выполняет fn в одной транзакции. Postgres кладёт tx в ctx;
// репозитории ордера и outbox читают её оттуда. pgx.Tx в application не протаскивается.
type TxManager interface {
	WithinTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// OutboxStore пишет событие в transactional outbox (тот же tx, что и ордер).
type OutboxStore interface {
	Append(ctx context.Context, event OutboxEvent) error
}

// OutboxRelayStore — очередь для relay. ClaimBatch должен вызываться внутри WithinTx:
// postgres держит строки через SELECT … FOR UPDATE SKIP LOCKED до commit/rollback.
type OutboxRelayStore interface {
	ClaimBatch(ctx context.Context, limit int) ([]OutboxEvent, error)
	MarkProcessed(ctx context.Context, ids []string) error
	RecordAttempt(ctx context.Context, id string) error
}

// OrderNotifier публикует обновления статуса ордера в in-memory hub.
type OrderNotifier interface {
	Publish(event UpdateEvent)
}

// OrderEventPublisher отправляет outbox-событие в брокер (G4).
// nil в relay = локальный hub без Kafka (тесты). При заданном publisher hub кормит consumer (G5).
type OrderEventPublisher interface {
	Publish(ctx context.Context, event OutboxEvent) error
}

// InboxStore дедуплицирует входящие команды по event_id / command_id (G6).
// InsertIfNew в том же tx, что UpdateStatus и outbox.Append.
// inserted=false — дубль, обработку пропускаем.
type InboxStore interface {
	InsertIfNew(ctx context.Context, eventID string) (inserted bool, err error)
}

// OrderCommandHandler применяет команду смены статуса (matching-engine / Kafka).
type OrderCommandHandler interface {
	Handle(ctx context.Context, cmd OrderCommand) error
}

// DeadLetterSink принимает яд (невалидный payload, NotFound, InvalidArgument)
// и сообщения, которые исчерпали ретраи, чтобы consumer не блокировал партицию.
type DeadLetterSink interface {
	Publish(ctx context.Context, original []byte, reason string) error
}

// OrderUpdateHub — pub/sub обновлений ордеров для streaming.
//
// Контракт: Subscribe/SubscribeUser возвращают канал и unsubscribe.
// Вызывающий (gRPC-стрим / use case) обязан вызвать unsubscribe при завершении.
// Hub канал не закрывает: Publish мог уже скопировать ch, send в закрытый канал паникует.
// Publish безопасен при одновременной отписке.
type OrderUpdateHub interface {
	OrderNotifier
	Subscribe(orderID string) (<-chan UpdateEvent, func())
	SubscribeUser(userID string, filter UserStreamFilter) (<-chan UpdateEvent, func())
}
