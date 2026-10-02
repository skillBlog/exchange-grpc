package application_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/exchange-grpc/orderservice/internal/application"
	"github.com/exchange-grpc/orderservice/internal/domain"
	"github.com/exchange-grpc/orderservice/internal/infrastructure/memory"
	"github.com/google/uuid"
)

func TestParseOrderCommand_acceptsEventIDAlias(t *testing.T) {
	eventID := uuid.Must(uuid.NewV7()).String()
	raw, err := json.Marshal(map[string]string{
		"event_id": eventID,
		"order_id": "11111111-1111-4111-8111-111111111111",
		"user_id":  "22222222-2222-4222-8222-222222222222",
		"status":   "filled",
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	cmd, err := application.ParseOrderCommand(raw)
	if err != nil {
		t.Fatalf("ParseOrderCommand() error = %v", err)
	}
	if cmd.CommandID != eventID {
		t.Fatalf("CommandID = %q", cmd.CommandID)
	}
	if cmd.Status != domain.OrderStatusFilled {
		t.Fatalf("Status = %q", cmd.Status)
	}
}

func TestParseOrderCommand_invalidJSON(t *testing.T) {
	_, err := application.ParseOrderCommand([]byte(`{`))
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("error = %v, want ErrInvalidArgument", err)
	}
}

func TestParseOrderCommand_invalidOrderID(t *testing.T) {
	raw := []byte(`{"command_id":"11111111-1111-4111-8111-111111111111","order_id":"not-a-uuid","user_id":"22222222-2222-4222-8222-222222222222","status":"filled"}`)
	_, err := application.ParseOrderCommand(raw)
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("error = %v, want ErrInvalidArgument", err)
	}
}

func TestApplyOrderCommand_duplicateSkipsSecondUpdate(t *testing.T) {
	repo := memory.NewOrderRepository()
	now := time.Now().UTC()
	order, err := domain.NewOrder(domain.NewOrderID(), "11111111-1111-1111-1111-111111111111", "BTC-USDT", domain.OrderSideBuy, domain.Money{}, mustDecimal(t, "1"), now)
	if err != nil {
		t.Fatalf("NewOrder() error = %v", err)
	}
	if err := repo.Create(context.Background(), order); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	outbox := &recordingOutbox{}
	update := application.NewUpdateOrderStatus(repo, memory.NewTxManager(), outbox, nil)
	uc := application.NewApplyOrderCommand(memory.NewTxManager(), memory.NewInboxStore(), update)
	cmd := application.OrderCommand{
		CommandID: uuid.Must(uuid.NewV7()).String(),
		OrderID:   order.ID,
		UserID:    "11111111-1111-1111-1111-111111111111",
		Status:    domain.OrderStatusFilled,
	}

	if err := uc.Execute(context.Background(), cmd); err != nil {
		t.Fatalf("first Execute() error = %v", err)
	}
	if err := uc.Execute(context.Background(), cmd); err != nil {
		t.Fatalf("duplicate Execute() error = %v", err)
	}
	if len(outbox.events) != 1 {
		t.Fatalf("appended %d events, want 1", len(outbox.events))
	}

	saved, err := repo.GetByID(context.Background(), order.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if saved.Status != domain.OrderStatusFilled {
		t.Fatalf("status = %q, want filled", saved.Status)
	}
}

func TestApplyOrderCommand_sameStatusStillRecordsInbox(t *testing.T) {
	repo := memory.NewOrderRepository()
	now := time.Now().UTC()
	order, err := domain.NewOrder(domain.NewOrderID(), "11111111-1111-1111-1111-111111111111", "BTC-USDT", domain.OrderSideBuy, domain.Money{}, mustDecimal(t, "1"), now)
	if err != nil {
		t.Fatalf("NewOrder() error = %v", err)
	}
	order.Status = domain.OrderStatusFilled
	order.UpdatedAt = now
	if err := repo.Create(context.Background(), order); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	outbox := &recordingOutbox{}
	update := application.NewUpdateOrderStatus(repo, memory.NewTxManager(), outbox, nil)
	uc := application.NewApplyOrderCommand(memory.NewTxManager(), memory.NewInboxStore(), update)
	cmd := application.OrderCommand{
		CommandID: uuid.Must(uuid.NewV7()).String(),
		OrderID:   order.ID,
		UserID:    "11111111-1111-1111-1111-111111111111",
		Status:    domain.OrderStatusFilled,
	}
	if err := uc.Execute(context.Background(), cmd); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if err := uc.Execute(context.Background(), cmd); err != nil {
		t.Fatalf("duplicate Execute() error = %v", err)
	}
	if len(outbox.events) != 0 {
		t.Fatalf("appended %d events, want 0 (same-status no-op)", len(outbox.events))
	}
}

type recordingDLQ struct {
	payloads [][]byte
	reasons  []string
	err      error
}

func (d *recordingDLQ) Publish(_ context.Context, original []byte, reason string) error {
	if d.err != nil {
		return d.err
	}
	copied := make([]byte, len(original))
	copy(copied, original)
	d.payloads = append(d.payloads, copied)
	d.reasons = append(d.reasons, reason)
	return nil
}

type stubCommandHandler struct {
	calls int
	err   error
}

func (h *stubCommandHandler) Handle(context.Context, application.OrderCommand) error {
	h.calls++
	return h.err
}

type flakyCommandHandler struct {
	fails int
	calls int
	err   error
}

func (h *flakyCommandHandler) Handle(context.Context, application.OrderCommand) error {
	h.calls++
	if h.calls <= h.fails {
		return h.err
	}
	return nil
}

func validCommandJSON(t *testing.T) []byte {
	t.Helper()
	raw, err := json.Marshal(map[string]string{
		"command_id": uuid.Must(uuid.NewV7()).String(),
		"order_id":   "11111111-1111-4111-8111-111111111111",
		"user_id":    "22222222-2222-4222-8222-222222222222",
		"status":     "filled",
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return raw
}

func TestProcessCommandMessage_invalidPayloadGoesToDLQ(t *testing.T) {
	dlq := &recordingDLQ{}
	handler := &stubCommandHandler{}
	if err := application.ProcessCommandMessage(context.Background(), []byte(`{`), handler, dlq, 3); err != nil {
		t.Fatalf("ProcessCommandMessage() error = %v", err)
	}
	if handler.calls != 0 {
		t.Fatalf("handler calls = %d, want 0", handler.calls)
	}
	if len(dlq.payloads) != 1 {
		t.Fatalf("dlq = %d, want 1", len(dlq.payloads))
	}
}

func TestProcessCommandMessage_notFoundGoesToDLQOnce(t *testing.T) {
	dlq := &recordingDLQ{}
	handler := &stubCommandHandler{err: domain.ErrNotFound}
	if err := application.ProcessCommandMessage(context.Background(), validCommandJSON(t), handler, dlq, 5); err != nil {
		t.Fatalf("ProcessCommandMessage() error = %v", err)
	}
	if handler.calls != 1 {
		t.Fatalf("handler calls = %d, want 1", handler.calls)
	}
	if len(dlq.payloads) != 1 {
		t.Fatalf("dlq = %d, want 1", len(dlq.payloads))
	}
}

func TestProcessCommandMessage_retriesThenSucceeds(t *testing.T) {
	dlq := &recordingDLQ{}
	handler := &flakyCommandHandler{fails: 2, err: domain.ErrConflict}
	if err := application.ProcessCommandMessage(context.Background(), validCommandJSON(t), handler, dlq, 5); err != nil {
		t.Fatalf("ProcessCommandMessage() error = %v", err)
	}
	if handler.calls != 3 {
		t.Fatalf("handler calls = %d, want 3", handler.calls)
	}
	if len(dlq.payloads) != 0 {
		t.Fatalf("dlq = %d, want 0", len(dlq.payloads))
	}
}

func TestProcessCommandMessage_exhaustedRetriesGoToDLQ(t *testing.T) {
	dlq := &recordingDLQ{}
	handler := &stubCommandHandler{err: domain.ErrConflict}
	if err := application.ProcessCommandMessage(context.Background(), validCommandJSON(t), handler, dlq, 3); err != nil {
		t.Fatalf("ProcessCommandMessage() error = %v", err)
	}
	if handler.calls != 3 {
		t.Fatalf("handler calls = %d, want 3", handler.calls)
	}
	if len(dlq.payloads) != 1 {
		t.Fatalf("dlq = %d, want 1", len(dlq.payloads))
	}
}

func TestProcessCommandMessage_dlqFailureDoesNotCommit(t *testing.T) {
	dlq := &recordingDLQ{err: errors.New("dlq down")}
	err := application.ProcessCommandMessage(context.Background(), []byte(`{`), &stubCommandHandler{}, dlq, 1)
	if err == nil {
		t.Fatal("expected error when DLQ publish fails")
	}
}

func TestIsPermanentCommandError(t *testing.T) {
	if !application.IsPermanentCommandError(domain.ErrInvalidArgument) {
		t.Fatal("InvalidArgument should be permanent")
	}
	if !application.IsPermanentCommandError(domain.ErrNotFound) {
		t.Fatal("NotFound should be permanent")
	}
	if application.IsPermanentCommandError(domain.ErrConflict) {
		t.Fatal("Conflict should be retryable")
	}
}
