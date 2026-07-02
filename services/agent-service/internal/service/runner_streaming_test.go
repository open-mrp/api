package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"

	"github.com/augno/api/shared/contracts"
	"github.com/augno/api/shared/messaging"
)

// mockBroker is a test double for messaging.MessageBroker that captures published messages.
type mockBroker struct {
	mu       sync.Mutex
	messages []publishedMessage
}

type publishedMessage struct {
	Exchange   string
	RoutingKey string
	Message    contracts.AmqpMessage
}

func (b *mockBroker) PublishMessage(_ context.Context, exchange, routingKey string, message contracts.AmqpMessage) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.messages = append(b.messages, publishedMessage{
		Exchange:   exchange,
		RoutingKey: routingKey,
		Message:    message,
	})
	return nil
}

func (b *mockBroker) ConsumeMessages(_ context.Context, _ string, _ messaging.MessageHandler, _ ...messaging.ConsumeOption) error {
	return nil
}

func (b *mockBroker) ConsumeFanout(_ context.Context, _ string, _ []string, _ messaging.MessageHandler, _ ...messaging.ConsumeOption) error {
	return nil
}

func (b *mockBroker) IsReady() bool { return true }
func (b *mockBroker) Close()        {}

// Ensure mockBroker satisfies the interface at compile time.
var _ messaging.MessageBroker = (*mockBroker)(nil)

func TestEmitDelta_PublishesCorrectPayload(t *testing.T) {
	t.Parallel()
	broker := &mockBroker{}

	svc := &runnerSvc{
		broker: broker,
	}

	ctx := context.Background()
	runID := "agr_test_run_123"
	accountID := "acct_test_456"
	deltaSeq := 7
	content := "Here is some streamed content."

	svc.emitDelta(ctx, runID, accountID, deltaSeq, content)

	// Verify broker was called exactly once
	broker.mu.Lock()
	defer broker.mu.Unlock()

	if len(broker.messages) != 1 {
		t.Fatalf("expected 1 published message, got %d", len(broker.messages))
	}

	msg := broker.messages[0]

	// Verify exchange
	if msg.Exchange != messaging.ApplicationExchange {
		t.Errorf("expected exchange %q, got %q", messaging.ApplicationExchange, msg.Exchange)
	}

	// Verify routing key
	expectedRoutingKey := string(contracts.AgentEventRunStep)
	if msg.RoutingKey != expectedRoutingKey {
		t.Errorf("expected routing key %q, got %q", expectedRoutingKey, msg.RoutingKey)
	}

	// Deserialize and verify payload
	var stepData messaging.AgentRunStepData
	if err := json.Unmarshal(msg.Message.Data, &stepData); err != nil {
		t.Fatalf("failed to unmarshal step data: %v", err)
	}

	if stepData.StepType != "content_delta" {
		t.Errorf("expected StepType 'content_delta', got %q", stepData.StepType)
	}

	expectedEventID := fmt.Sprintf("delta-%s-%d", runID, deltaSeq)
	if stepData.EventID != expectedEventID {
		t.Errorf("expected EventID %q, got %q", expectedEventID, stepData.EventID)
	}

	if stepData.AgentRunID != runID {
		t.Errorf("expected AgentRunID %q, got %q", runID, stepData.AgentRunID)
	}

	if stepData.AccountID != accountID {
		t.Errorf("expected AccountID %q, got %q", accountID, stepData.AccountID)
	}

	if stepData.Content == nil || *stepData.Content != content {
		t.Errorf("expected Content %q, got %v", content, stepData.Content)
	}

	if stepData.Sequence != deltaSeq {
		t.Errorf("expected Sequence %d, got %d", deltaSeq, stepData.Sequence)
	}

	if stepData.Title != "Content delta" {
		t.Errorf("expected Title 'Content delta', got %q", stepData.Title)
	}

	if stepData.CreatedAt == "" {
		t.Error("expected non-empty CreatedAt")
	}
}
