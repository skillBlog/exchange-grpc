package config_test

import (
	"testing"
	"time"

	"github.com/exchange-grpc/orderservice/pkg/config"
)

func TestLoadConfig_kafkaBrokersCSV(t *testing.T) {
	t.Setenv("KAFKA_BROKERS", "kafka:19092, localhost:9092")
	t.Setenv("KAFKA_ORDER_TOPIC", "order.events")
	t.Setenv("KAFKA_PRODUCE_TIMEOUT", "2s")

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if len(cfg.KafkaBrokers) != 2 || cfg.KafkaBrokers[0] != "kafka:19092" || cfg.KafkaBrokers[1] != "localhost:9092" {
		t.Fatalf("KafkaBrokers = %#v", cfg.KafkaBrokers)
	}
	if cfg.KafkaOrderTopic != "order.events" {
		t.Fatalf("KafkaOrderTopic = %q", cfg.KafkaOrderTopic)
	}
	if cfg.KafkaProduceTimeout != 2*time.Second {
		t.Fatalf("KafkaProduceTimeout = %v", cfg.KafkaProduceTimeout)
	}
	if cfg.KafkaCommandTopic != "order.commands" {
		t.Fatalf("default command topic = %q", cfg.KafkaCommandTopic)
	}
	if cfg.KafkaCommandDLQTopic != "order.commands.dlq" {
		t.Fatalf("default dlq topic = %q", cfg.KafkaCommandDLQTopic)
	}
	if cfg.KafkaCommandGroup != "orderservice-commands" {
		t.Fatalf("default command group = %q", cfg.KafkaCommandGroup)
	}
}

func TestLoadConfig_emptyKafkaBrokers(t *testing.T) {
	t.Setenv("KAFKA_BROKERS", "")

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if len(cfg.KafkaBrokers) != 0 {
		t.Fatalf("KafkaBrokers = %#v, want empty", cfg.KafkaBrokers)
	}
	if cfg.KafkaOrderTopic != "order.events" {
		t.Fatalf("default topic = %q", cfg.KafkaOrderTopic)
	}
	if cfg.KafkaCommandTopic != "order.commands" {
		t.Fatalf("default command topic = %q", cfg.KafkaCommandTopic)
	}
}

func TestLoadConfig_commandTopics(t *testing.T) {
	t.Setenv("KAFKA_COMMAND_TOPIC", "cmd")
	t.Setenv("KAFKA_COMMAND_DLQ_TOPIC", "cmd.dlq")
	t.Setenv("KAFKA_COMMAND_GROUP", "orderservice-commands")
	t.Setenv("KAFKA_COMMAND_MAX_ATTEMPTS", "7")

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if cfg.KafkaCommandTopic != "cmd" {
		t.Fatalf("KafkaCommandTopic = %q", cfg.KafkaCommandTopic)
	}
	if cfg.KafkaCommandDLQTopic != "cmd.dlq" {
		t.Fatalf("KafkaCommandDLQTopic = %q", cfg.KafkaCommandDLQTopic)
	}
	if cfg.KafkaCommandMaxAttempts != 7 {
		t.Fatalf("KafkaCommandMaxAttempts = %d", cfg.KafkaCommandMaxAttempts)
	}
}
