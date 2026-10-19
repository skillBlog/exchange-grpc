package kafka

import (
	"context"

	kafkago "github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel"
)

// headerCarrier переносит W3C traceparent/tracestate в заголовки Kafka.
type headerCarrier struct {
	headers []kafkago.Header
}

func (c *headerCarrier) Get(key string) string {
	for _, header := range c.headers {
		if header.Key == key {
			return string(header.Value)
		}
	}
	return ""
}

func (c *headerCarrier) Set(key, value string) {
	for i, header := range c.headers {
		if header.Key == key {
			c.headers[i].Value = []byte(value)
			return
		}
	}
	c.headers = append(c.headers, kafkago.Header{Key: key, Value: []byte(value)})
}

func (c *headerCarrier) Keys() []string {
	keys := make([]string, len(c.headers))
	for i, header := range c.headers {
		keys[i] = header.Key
	}
	return keys
}

func injectTrace(ctx context.Context, headers []kafkago.Header) []kafkago.Header {
	if ctx == nil {
		return headers
	}
	carrier := &headerCarrier{headers: headers}
	otel.GetTextMapPropagator().Inject(ctx, carrier)
	return carrier.headers
}

func extractTrace(ctx context.Context, headers []kafkago.Header) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return otel.GetTextMapPropagator().Extract(ctx, &headerCarrier{headers: headers})
}
