package redisbus

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/zap"
)

func TestReadMessages_badPayloadDoesNotStopNext(t *testing.T) {
	payloads := []string{"bad", "good"}
	nextIndex := 0
	var got []string
	err := readMessages(context.Background(), func(context.Context) (string, error) {
		if nextIndex >= len(payloads) {
			return "", context.Canceled
		}
		payload := payloads[nextIndex]
		nextIndex++
		return payload, nil
	}, func(payload []byte) error {
		got = append(got, string(payload))
		if string(payload) == "bad" {
			return errors.New("decode")
		}
		return nil
	}, zap.NewNop())
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("readMessages() error = %v, want context.Canceled", err)
	}
	if len(got) != 2 || got[0] != "bad" || got[1] != "good" {
		t.Fatalf("delivered = %#v", got)
	}
}
