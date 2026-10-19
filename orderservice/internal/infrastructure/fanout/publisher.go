package fanout

import (
	"context"
	"fmt"

	"github.com/exchange-grpc/orderservice/internal/application"
	orderkafka "github.com/exchange-grpc/orderservice/internal/infrastructure/kafka"
)

// Chain вызывает publisher-ы по порядку и останавливается на первой ошибке.
type Chain struct {
	pubs []application.OrderEventPublisher
}

// New собирает цепочку. nil-элементы пропускаются.
func New(pubs ...application.OrderEventPublisher) *Chain {
	return &Chain{pubs: pubs}
}

// Publish отправляет событие в каждый следующий publisher.
func (c *Chain) Publish(ctx context.Context, event application.OutboxEvent) error {
	if c == nil {
		return fmt.Errorf("order event fanout is not configured")
	}
	for _, pub := range c.pubs {
		if pub == nil {
			continue
		}
		if err := pub.Publish(ctx, event); err != nil {
			return err
		}
	}
	return nil
}

// HubPublisher пишет событие в локальный hub. Повтор event_id не публикуется второй раз.
type HubPublisher struct {
	forward *orderkafka.HubForwarder
}

// NewHubPublisher создаёт publisher в hub одного процесса.
func NewHubPublisher(hub application.OrderNotifier) *HubPublisher {
	return &HubPublisher{forward: orderkafka.NewHubForwarder(hub)}
}

// Publish сериализует конверт и отдаёт его hub.
func (p *HubPublisher) Publish(_ context.Context, event application.OutboxEvent) error {
	if p == nil || p.forward == nil {
		return fmt.Errorf("hub publisher is not configured")
	}
	body, err := orderkafka.MarshalEnvelope(event)
	if err != nil {
		return err
	}
	return p.forward.Deliver(body)
}

var (
	_ application.OrderEventPublisher = (*Chain)(nil)
	_ application.OrderEventPublisher = (*HubPublisher)(nil)
)
