package messaging

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/augno/api/shared/contracts"
	"github.com/go-sql-driver/mysql"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/suite"
)

// ---------------------------------------------------------------------------
// mock InboxRepo
// ---------------------------------------------------------------------------

type mockInboxRepo struct {
	tryInsertFn            func(ctx context.Context, input InboxRecordInput) (int64, error)
	getByMessageAndHandler func(ctx context.Context, messageID, handler string) (*InboxRecord, error)
	markProcessedFn        func(ctx context.Context, id int64) error
	markFailedFn           func(ctx context.Context, id int64, errMsg string) error
}

func (m *mockInboxRepo) TryInsert(ctx context.Context, input InboxRecordInput) (int64, error) {
	if m.tryInsertFn != nil {
		return m.tryInsertFn(ctx, input)
	}
	return 1, nil
}

func (m *mockInboxRepo) GetByMessageAndHandler(ctx context.Context, messageID, handler string) (*InboxRecord, error) {
	if m.getByMessageAndHandler != nil {
		return m.getByMessageAndHandler(ctx, messageID, handler)
	}
	return nil, errors.New("not found")
}

func (m *mockInboxRepo) MarkProcessed(ctx context.Context, id int64) error {
	if m.markProcessedFn != nil {
		return m.markProcessedFn(ctx, id)
	}
	return nil
}

func (m *mockInboxRepo) MarkFailed(ctx context.Context, id int64, errMsg string) error {
	if m.markFailedFn != nil {
		return m.markFailedFn(ctx, id, errMsg)
	}
	return nil
}

var _ InboxRepo = (*mockInboxRepo)(nil)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func mysqlDupError() error {
	return &mysql.MySQLError{Number: 1062, Message: "Duplicate entry"}
}

func deliveryWithMessageID(messageID string) amqp.Delivery {
	body, _ := json.Marshal(contracts.AmqpMessage{
		MessageID: messageID,
	})
	return amqp.Delivery{
		MessageId: messageID,
		Body:      body,
	}
}

func deliveryWithBodyMessageID(bodyMessageID string) amqp.Delivery {
	body, _ := json.Marshal(contracts.AmqpMessage{
		MessageID: bodyMessageID,
	})
	return amqp.Delivery{
		// MessageId header is empty — should fall back to body
		Body: body,
	}
}

func deliveryWithoutMessageID() amqp.Delivery {
	body, _ := json.Marshal(contracts.AmqpMessage{})
	return amqp.Delivery{Body: body}
}

// ---------------------------------------------------------------------------
// suite
// ---------------------------------------------------------------------------

type InboxConsumerTestSuite struct {
	suite.Suite
}

// ---------------------------------------------------------------------------
// tests
// ---------------------------------------------------------------------------

func (s *InboxConsumerTestSuite) TestWrap_NewMessage_ProcessedSuccessfully() {
	var markProcessedCalled bool
	repo := &mockInboxRepo{
		tryInsertFn: func(_ context.Context, input InboxRecordInput) (int64, error) {
			s.Equal("msg_1", input.MessageID)
			s.Equal("test-handler", input.Handler)
			return 42, nil
		},
		markProcessedFn: func(_ context.Context, id int64) error {
			s.Equal(int64(42), id)
			markProcessedCalled = true
			return nil
		},
	}

	consumer := NewInboxConsumer(repo, "test-service")
	handlerCalled := false
	wrapped := consumer.Wrap("test-handler", func(_ context.Context, _ amqp.Delivery) error {
		handlerCalled = true
		return nil
	})

	err := wrapped(context.Background(), deliveryWithMessageID("msg_1"))
	s.NoError(err)
	s.True(handlerCalled)
	s.True(markProcessedCalled)
}

func (s *InboxConsumerTestSuite) TestWrap_NewMessage_HandlerFails() {
	var markFailedID int64
	var markFailedMsg string
	repo := &mockInboxRepo{
		tryInsertFn: func(_ context.Context, _ InboxRecordInput) (int64, error) {
			return 42, nil
		},
		markFailedFn: func(_ context.Context, id int64, errMsg string) error {
			markFailedID = id
			markFailedMsg = errMsg
			return nil
		},
	}

	consumer := NewInboxConsumer(repo, "test-service")
	wrapped := consumer.Wrap("test-handler", func(_ context.Context, _ amqp.Delivery) error {
		return errors.New("handler exploded")
	})

	err := wrapped(context.Background(), deliveryWithMessageID("msg_1"))
	s.Error(err)
	s.Equal("handler exploded", err.Error())
	s.Equal(int64(42), markFailedID)
	s.Equal("handler exploded", markFailedMsg)
}

func (s *InboxConsumerTestSuite) TestWrap_DuplicateMessage_AlreadyProcessed() {
	repo := &mockInboxRepo{
		tryInsertFn: func(_ context.Context, _ InboxRecordInput) (int64, error) {
			return 0, mysqlDupError()
		},
		getByMessageAndHandler: func(_ context.Context, _, _ string) (*InboxRecord, error) {
			return &InboxRecord{
				ID:     1,
				Status: InboxStatusProcessed,
			}, nil
		},
	}

	consumer := NewInboxConsumer(repo, "test-service")
	handlerCalled := false
	wrapped := consumer.Wrap("test-handler", func(_ context.Context, _ amqp.Delivery) error {
		handlerCalled = true
		return nil
	})

	err := wrapped(context.Background(), deliveryWithMessageID("msg_dup"))
	s.NoError(err)
	s.False(handlerCalled) // handler should NOT be called
}

func (s *InboxConsumerTestSuite) TestWrap_DuplicateMessage_PreviouslyFailed_Retries() {
	lastErr := "previous failure"
	repo := &mockInboxRepo{
		tryInsertFn: func(_ context.Context, _ InboxRecordInput) (int64, error) {
			return 0, mysqlDupError()
		},
		getByMessageAndHandler: func(_ context.Context, _, _ string) (*InboxRecord, error) {
			return &InboxRecord{
				ID:        1,
				Status:    InboxStatusReceived,
				LastError: &lastErr,
				Attempts:  1,
			}, nil
		},
	}

	consumer := NewInboxConsumer(repo, "test-service")
	handlerCalled := false
	wrapped := consumer.Wrap("test-handler", func(_ context.Context, _ amqp.Delivery) error {
		handlerCalled = true
		return nil
	})

	err := wrapped(context.Background(), deliveryWithMessageID("msg_retry"))
	s.NoError(err)
	s.True(handlerCalled) // handler should be retried
}

func (s *InboxConsumerTestSuite) TestWrap_DuplicateMessage_CrashRecovery_Retries() {
	repo := &mockInboxRepo{
		tryInsertFn: func(_ context.Context, _ InboxRecordInput) (int64, error) {
			return 0, mysqlDupError()
		},
		getByMessageAndHandler: func(_ context.Context, _, _ string) (*InboxRecord, error) {
			return &InboxRecord{
				ID:       1,
				Status:   InboxStatusReceived,
				Attempts: 0,
				// LastError is nil — crash recovery scenario
			}, nil
		},
	}

	consumer := NewInboxConsumer(repo, "test-service")
	handlerCalled := false
	wrapped := consumer.Wrap("test-handler", func(_ context.Context, _ amqp.Delivery) error {
		handlerCalled = true
		return nil
	})

	err := wrapped(context.Background(), deliveryWithMessageID("msg_crash"))
	s.NoError(err)
	s.True(handlerCalled) // should be re-processed
}

func (s *InboxConsumerTestSuite) TestWrap_NoMessageID_ProcessesWithoutDedup() {
	repo := &mockInboxRepo{
		tryInsertFn: func(_ context.Context, _ InboxRecordInput) (int64, error) {
			s.Fail("TryInsert should not be called when there is no message ID")
			return 0, nil
		},
	}

	consumer := NewInboxConsumer(repo, "test-service")
	handlerCalled := false
	wrapped := consumer.Wrap("test-handler", func(_ context.Context, _ amqp.Delivery) error {
		handlerCalled = true
		return nil
	})

	err := wrapped(context.Background(), deliveryWithoutMessageID())
	s.NoError(err)
	s.True(handlerCalled) // handler runs directly without dedup
}

func (s *InboxConsumerTestSuite) TestWrap_MessageIDFromBody() {
	repo := &mockInboxRepo{
		tryInsertFn: func(_ context.Context, input InboxRecordInput) (int64, error) {
			s.Equal("body_msg_id", input.MessageID)
			return 1, nil
		},
	}

	consumer := NewInboxConsumer(repo, "test-service")
	wrapped := consumer.Wrap("test-handler", func(_ context.Context, _ amqp.Delivery) error {
		return nil
	})

	err := wrapped(context.Background(), deliveryWithBodyMessageID("body_msg_id"))
	s.NoError(err)
}

func (s *InboxConsumerTestSuite) TestWrap_TryInsertNonDuplicateError_ProcessesAnyway() {
	repo := &mockInboxRepo{
		tryInsertFn: func(_ context.Context, _ InboxRecordInput) (int64, error) {
			return 0, errors.New("connection refused")
		},
	}

	consumer := NewInboxConsumer(repo, "test-service")
	handlerCalled := false
	wrapped := consumer.Wrap("test-handler", func(_ context.Context, _ amqp.Delivery) error {
		handlerCalled = true
		return nil
	})

	err := wrapped(context.Background(), deliveryWithMessageID("msg_1"))
	s.NoError(err)
	s.True(handlerCalled) // graceful degradation
}

func (s *InboxConsumerTestSuite) TestWrap_DuplicateFetchError_SkipsMessage() {
	repo := &mockInboxRepo{
		tryInsertFn: func(_ context.Context, _ InboxRecordInput) (int64, error) {
			return 0, mysqlDupError()
		},
		getByMessageAndHandler: func(_ context.Context, _, _ string) (*InboxRecord, error) {
			return nil, errors.New("db unavailable")
		},
	}

	consumer := NewInboxConsumer(repo, "test-service")
	handlerCalled := false
	wrapped := consumer.Wrap("test-handler", func(_ context.Context, _ amqp.Delivery) error {
		handlerCalled = true
		return nil
	})

	err := wrapped(context.Background(), deliveryWithMessageID("msg_dup"))
	s.NoError(err) // ACKed to prevent infinite redelivery
	s.False(handlerCalled)
}

func (s *InboxConsumerTestSuite) TestWrap_MarkProcessedError_StillReturnsNil() {
	repo := &mockInboxRepo{
		tryInsertFn: func(_ context.Context, _ InboxRecordInput) (int64, error) {
			return 42, nil
		},
		markProcessedFn: func(_ context.Context, _ int64) error {
			return errors.New("mark failed")
		},
	}

	consumer := NewInboxConsumer(repo, "test-service")
	wrapped := consumer.Wrap("test-handler", func(_ context.Context, _ amqp.Delivery) error {
		return nil
	})

	err := wrapped(context.Background(), deliveryWithMessageID("msg_1"))
	s.NoError(err) // handler succeeded, even though MarkProcessed failed
}

func (s *InboxConsumerTestSuite) TestWrap_MarkFailedError_StillReturnsHandlerError() {
	repo := &mockInboxRepo{
		tryInsertFn: func(_ context.Context, _ InboxRecordInput) (int64, error) {
			return 42, nil
		},
		markFailedFn: func(_ context.Context, _ int64, _ string) error {
			return errors.New("mark failed error")
		},
	}

	consumer := NewInboxConsumer(repo, "test-service")
	wrapped := consumer.Wrap("test-handler", func(_ context.Context, _ amqp.Delivery) error {
		return errors.New("original handler error")
	})

	err := wrapped(context.Background(), deliveryWithMessageID("msg_1"))
	s.Error(err)
	s.Equal("original handler error", err.Error()) // returns original error, not mark failed error
}

// ---------------------------------------------------------------------------
// runner
// ---------------------------------------------------------------------------

func TestInboxConsumerTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(InboxConsumerTestSuite))
}
