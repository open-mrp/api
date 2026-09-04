package messaging

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	apierror "github.com/open-mrp/api/shared/errors"

	"github.com/go-sql-driver/mysql"
	"github.com/open-mrp/api/shared/contracts"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/suite"
)

// ---------------------------------------------------------------------------
// mock InboxRepo
// ---------------------------------------------------------------------------

type mockInboxRepo struct {
	tryInsertFn            func(ctx context.Context, input InboxRecordInput) (int64, error)
	getByMessageAndHandler func(ctx context.Context, messageID, handler string) (*InboxRecord, error)
	claimFn                func(ctx context.Context, id int64, owner string, ttlSeconds int) (bool, error)
	completeFn             func(ctx context.Context, id int64, owner string) (bool, error)
	markFailedFn           func(ctx context.Context, id int64, owner, errMsg string) error
	markDiscardedFn        func(ctx context.Context, id int64, owner, reason string) error
	markIgnoredFn          func(ctx context.Context, id int64, owner, reason string) error
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

func (m *mockInboxRepo) Claim(ctx context.Context, id int64, owner string, ttlSeconds int) (bool, error) {
	if m.claimFn != nil {
		return m.claimFn(ctx, id, owner, ttlSeconds)
	}
	return true, nil
}

func (m *mockInboxRepo) Complete(ctx context.Context, id int64, owner string) (bool, error) {
	if m.completeFn != nil {
		return m.completeFn(ctx, id, owner)
	}
	return true, nil
}

func (m *mockInboxRepo) MarkDiscarded(ctx context.Context, id int64, owner, reason string) error {
	if m.markDiscardedFn != nil {
		return m.markDiscardedFn(ctx, id, owner, reason)
	}
	return nil
}

func (m *mockInboxRepo) MarkIgnored(ctx context.Context, id int64, owner, reason string) error {
	if m.markIgnoredFn != nil {
		return m.markIgnoredFn(ctx, id, owner, reason)
	}
	return nil
}

func (m *mockInboxRepo) MarkFailed(ctx context.Context, id int64, owner, errMsg string) error {
	if m.markFailedFn != nil {
		return m.markFailedFn(ctx, id, owner, errMsg)
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
		completeFn: func(_ context.Context, id int64, _ string) (bool, error) {
			s.Equal(int64(42), id)
			markProcessedCalled = true
			return true, nil
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
		markFailedFn: func(_ context.Context, id int64, _ string, errMsg string) error {
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

// An abandoned attempt — no lease, no recorded error — is retried, but only after this consumer takes the lease.
func (s *InboxConsumerTestSuite) TestWrap_DuplicateMessage_LeaseLapsed_ClaimsAndRetries() {
	var claimedID int64
	repo := &mockInboxRepo{
		tryInsertFn: func(_ context.Context, _ InboxRecordInput) (int64, error) {
			return 0, mysqlDupError()
		},
		getByMessageAndHandler: func(_ context.Context, _, _ string) (*InboxRecord, error) {
			expired := time.Now().Add(-time.Minute)
			return &InboxRecord{
				ID:            1,
				Status:        InboxStatusReceived,
				Attempts:      0,
				LockExpiresAt: &expired,
			}, nil
		},
		claimFn: func(_ context.Context, id int64, _ string, _ int) (bool, error) {
			claimedID = id
			return true, nil
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
	s.True(handlerCalled)
	s.Equal(int64(1), claimedID)
}

// The case the lease exists for: a redelivery arriving while the first attempt is still running must not run the handler alongside it. Re-invoking here is what applied a committed-but-unmarked message a second time.
func (s *InboxConsumerTestSuite) TestWrap_DuplicateMessage_LeaseHeld_DoesNotRunHandler() {
	repo := &mockInboxRepo{
		tryInsertFn: func(_ context.Context, _ InboxRecordInput) (int64, error) {
			return 0, mysqlDupError()
		},
		getByMessageAndHandler: func(_ context.Context, _, _ string) (*InboxRecord, error) {
			live := time.Now().Add(time.Minute)
			return &InboxRecord{
				ID:            1,
				Status:        InboxStatusReceived,
				LockOwner:     ptr("someone-else"),
				LockExpiresAt: &live,
			}, nil
		},
		claimFn: func(_ context.Context, _ int64, _ string, _ int) (bool, error) {
			s.Fail("Claim must not be attempted while the lease is live")
			return false, nil
		},
	}

	consumer := NewInboxConsumer(repo, "test-service")
	handlerCalled := false
	wrapped := consumer.Wrap("test-handler", func(_ context.Context, _ amqp.Delivery) error {
		handlerCalled = true
		return nil
	})

	err := wrapped(context.Background(), deliveryWithMessageID("msg_inflight"))
	s.ErrorIs(err, ErrInboxLeaseHeld)
	s.False(handlerCalled)
}

// Two consumers reaching the same lapsed record: the one that loses the conditional claim must not run.
func (s *InboxConsumerTestSuite) TestWrap_DuplicateMessage_ClaimLost_DoesNotRunHandler() {
	repo := &mockInboxRepo{
		tryInsertFn: func(_ context.Context, _ InboxRecordInput) (int64, error) {
			return 0, mysqlDupError()
		},
		getByMessageAndHandler: func(_ context.Context, _, _ string) (*InboxRecord, error) {
			return &InboxRecord{ID: 1, Status: InboxStatusReceived}, nil
		},
		claimFn: func(_ context.Context, _ int64, _ string, _ int) (bool, error) {
			return false, nil
		},
	}

	consumer := NewInboxConsumer(repo, "test-service")
	handlerCalled := false
	wrapped := consumer.Wrap("test-handler", func(_ context.Context, _ amqp.Delivery) error {
		handlerCalled = true
		return nil
	})

	err := wrapped(context.Background(), deliveryWithMessageID("msg_raced"))
	s.ErrorIs(err, ErrInboxLeaseHeld)
	s.False(handlerCalled)
}

// A discarded message is terminal: a redelivery must not run the handler again.
func (s *InboxConsumerTestSuite) TestWrap_DuplicateMessage_Discarded_Skips() {
	repo := &mockInboxRepo{
		tryInsertFn: func(_ context.Context, _ InboxRecordInput) (int64, error) {
			return 0, mysqlDupError()
		},
		getByMessageAndHandler: func(_ context.Context, _, _ string) (*InboxRecord, error) {
			return &InboxRecord{ID: 1, Status: InboxStatusDiscarded}, nil
		},
	}

	consumer := NewInboxConsumer(repo, "test-service")
	handlerCalled := false
	wrapped := consumer.Wrap("test-handler", func(_ context.Context, _ amqp.Delivery) error {
		handlerCalled = true
		return nil
	})

	err := wrapped(context.Background(), deliveryWithMessageID("msg_discarded"))
	s.NoError(err)
	s.False(handlerCalled)
}

// Discard records the reason and ACKs, rather than leaving the row looking like work that succeeded.
func (s *InboxConsumerTestSuite) TestWrap_Discard_RecordsReasonAndAcks() {
	var discardedID int64
	var reason string
	repo := &mockInboxRepo{
		tryInsertFn: func(_ context.Context, _ InboxRecordInput) (int64, error) { return 9, nil },
		markDiscardedFn: func(_ context.Context, id int64, _ string, r string) error {
			discardedID = id
			reason = r
			return nil
		},
		completeFn: func(_ context.Context, _ int64, _ string) (bool, error) {
			s.Fail("a discarded message must not also be marked processed")
			return false, nil
		},
		markFailedFn: func(_ context.Context, _ int64, _ string, _ string) error {
			s.Fail("a deliberate discard is not a failure to retry")
			return nil
		},
	}

	consumer := NewInboxConsumer(repo, "test-service")
	wrapped := consumer.Wrap("test-handler", func(ctx context.Context, _ amqp.Delivery) error {
		return consumer.Discard(ctx, "no account on event")
	})

	err := wrapped(context.Background(), deliveryWithMessageID("msg_malformed"))
	s.NoError(err)
	s.Equal(int64(9), discardedID)
	s.Equal("no account on event", reason)
}

// The handler sees the record id, which is what lets it commit its recovery point inside its own transaction.
func (s *InboxConsumerTestSuite) TestWrap_HandlerReceivesTheLease() {
	repo := &mockInboxRepo{
		tryInsertFn: func(_ context.Context, _ InboxRecordInput) (int64, error) { return 77, nil },
	}

	consumer := NewInboxConsumer(repo, "test-service")
	var lease InboxLease
	var ok bool
	wrapped := consumer.Wrap("test-handler", func(ctx context.Context, _ amqp.Delivery) error {
		lease, ok = InboxLeaseFromContext(ctx)
		return nil
	})

	s.NoError(wrapped(context.Background(), deliveryWithMessageID("msg_ctx")))
	s.True(ok)
	s.Equal(int64(77), lease.RecordID)
	// The owner travels with the record id: every write that closes the record out is conditional on it.
	s.NotEmpty(lease.Owner)
}

// A handler that rolled back because another attempt won the race reports no error: nothing failed and nothing is owed.
func (s *InboxConsumerTestSuite) TestWrap_AlreadyCompletedByConcurrentAttempt_Acks() {
	repo := &mockInboxRepo{
		tryInsertFn: func(_ context.Context, _ InboxRecordInput) (int64, error) { return 5, nil },
		markFailedFn: func(_ context.Context, _ int64, _ string, _ string) error {
			s.Fail("losing a completion race is not a failure")
			return nil
		},
	}

	consumer := NewInboxConsumer(repo, "test-service")
	wrapped := consumer.Wrap("test-handler", func(_ context.Context, _ amqp.Delivery) error {
		return fmt.Errorf("rolled back: %w", ErrInboxAlreadyCompleted)
	})

	s.NoError(wrapped(context.Background(), deliveryWithMessageID("msg_raced_complete")))
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
		completeFn: func(_ context.Context, _ int64, _ string) (bool, error) {
			return false, errors.New("mark failed")
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
		markFailedFn: func(_ context.Context, _ int64, _ string, _ string) error {
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

// A handler that fails must record WHAT failed, not merely that it did.
//
// Most APIError constructors set only a client-facing message, and APIError.Error() renders only the
// developer-facing InternalMessage — deliberately, so a nested error carrying nothing but a public
// message contributes nothing to its parent's text. The consequence here was that such a failure
// reached message_inbox.last_error as NULL: five recalc_item_burn_rate rows sat in e2e recording
// four attempts each and no reason for any of them, which is precisely the state nobody can triage.
func (s *InboxConsumerTestSuite) TestWrap_RecordsAReasonForAPublicMessageOnlyFailure() {
	var recorded string
	repo := &mockInboxRepo{
		tryInsertFn: func(_ context.Context, _ InboxRecordInput) (int64, error) { return 7, nil },
		markFailedFn: func(_ context.Context, _ int64, _ string, errMsg string) error {
			recorded = errMsg
			return nil
		},
	}

	consumer := NewInboxConsumer(repo, "test-service")
	wrapped := consumer.Wrap("test.handler", func(context.Context, amqp.Delivery) error {
		return apierror.NewValidationError("bad input")
	})

	err := wrapped(context.Background(), deliveryWithMessageID("msg_public_only"))

	s.Require().Error(err)
	s.NotEmpty(recorded, "a failure recorded with no reason is a row nobody can act on")
	s.Contains(recorded, "bad input")
}

func ptr(v string) *string { return &v }

// ---------------------------------------------------------------------------
// lease mechanics
// ---------------------------------------------------------------------------

// The lease a new record is inserted under has to identify this consumer and carry its duration, or nothing downstream can tell a live attempt from an abandoned one.
func (s *InboxConsumerTestSuite) TestWrap_NewMessage_InsertsUnderThisConsumersLease() {
	var gotOwner string
	var gotTTL int
	repo := &mockInboxRepo{
		tryInsertFn: func(_ context.Context, input InboxRecordInput) (int64, error) {
			gotOwner = input.LockOwner
			gotTTL = input.LockTTLSeconds
			return 1, nil
		},
	}

	consumer := NewInboxConsumer(repo, "test-service")
	wrapped := consumer.Wrap("test-handler", func(context.Context, amqp.Delivery) error { return nil })

	s.NoError(wrapped(context.Background(), deliveryWithMessageID("msg_lease")))
	s.NotEmpty(gotOwner, "a lease with no owner cannot be attributed to an attempt")
	s.Equal(DefaultInboxLeaseSeconds, gotTTL)
}

// Two consumers of the same queue must not share a lease owner, or each would treat the other's live lease as its own.
func (s *InboxConsumerTestSuite) TestNewInboxConsumer_MintsADistinctLeaseOwner() {
	var first, second string
	repoFor := func(dst *string) *mockInboxRepo {
		return &mockInboxRepo{tryInsertFn: func(_ context.Context, in InboxRecordInput) (int64, error) {
			*dst = in.LockOwner
			return 1, nil
		}}
	}

	a := NewInboxConsumer(repoFor(&first), "test-service")
	b := NewInboxConsumer(repoFor(&second), "test-service")
	s.NoError(a.Wrap("h", func(context.Context, amqp.Delivery) error { return nil })(context.Background(), deliveryWithMessageID("m1")))
	s.NoError(b.Wrap("h", func(context.Context, amqp.Delivery) error { return nil })(context.Background(), deliveryWithMessageID("m2")))

	s.NotEqual(first, second)
}

// Handlers that legitimately run for minutes raise the lease so a slow attempt is not mistaken for an abandoned one.
func (s *InboxConsumerTestSuite) TestWithLeaseSeconds_OverridesTheDefault() {
	var gotTTL int
	repo := &mockInboxRepo{
		tryInsertFn: func(_ context.Context, input InboxRecordInput) (int64, error) {
			gotTTL = input.LockTTLSeconds
			return 1, nil
		},
	}

	consumer := NewInboxConsumer(repo, "test-service").WithLeaseSeconds(1800)
	wrapped := consumer.Wrap("test-handler", func(context.Context, amqp.Delivery) error { return nil })

	s.NoError(wrapped(context.Background(), deliveryWithMessageID("msg_long")))
	s.Equal(1800, gotTTL)
}

// A nonsense override is ignored rather than disabling the lease outright.
func (s *InboxConsumerTestSuite) TestWithLeaseSeconds_IgnoresNonPositive() {
	consumer := NewInboxConsumer(&mockInboxRepo{}, "test-service").WithLeaseSeconds(0)
	s.Equal(DefaultInboxLeaseSeconds, consumer.leaseSeconds)

	consumer = NewInboxConsumer(&mockInboxRepo{}, "test-service").WithLeaseSeconds(-1)
	s.Equal(DefaultInboxLeaseSeconds, consumer.leaseSeconds)
}

// A claim that errors is not a license to run: the record's state is unknown, so the handler stays put.
func (s *InboxConsumerTestSuite) TestWrap_ClaimError_DoesNotRunHandler() {
	repo := &mockInboxRepo{
		tryInsertFn: func(context.Context, InboxRecordInput) (int64, error) { return 0, mysqlDupError() },
		getByMessageAndHandler: func(context.Context, string, string) (*InboxRecord, error) {
			return &InboxRecord{ID: 1, Status: InboxStatusReceived}, nil
		},
		claimFn: func(context.Context, int64, string, int) (bool, error) {
			return false, errors.New("db unavailable")
		},
	}

	consumer := NewInboxConsumer(repo, "test-service")
	handlerCalled := false
	wrapped := consumer.Wrap("test-handler", func(context.Context, amqp.Delivery) error {
		handlerCalled = true
		return nil
	})

	err := wrapped(context.Background(), deliveryWithMessageID("msg_claim_err"))
	s.Error(err)
	s.False(handlerCalled)
}

// The retry runs under the claiming consumer's own lease, not the dead attempt's.
func (s *InboxConsumerTestSuite) TestWrap_LeaseLapsed_ClaimsWithOwnOwnerAndTTL() {
	var claimOwner string
	var claimTTL int
	repo := &mockInboxRepo{
		tryInsertFn: func(context.Context, InboxRecordInput) (int64, error) { return 0, mysqlDupError() },
		getByMessageAndHandler: func(context.Context, string, string) (*InboxRecord, error) {
			stale := time.Now().Add(-time.Hour)
			return &InboxRecord{ID: 3, Status: InboxStatusReceived, LockOwner: ptr("dead-pod"), LockExpiresAt: &stale}, nil
		},
		claimFn: func(_ context.Context, _ int64, owner string, ttl int) (bool, error) {
			claimOwner = owner
			claimTTL = ttl
			return true, nil
		},
	}

	consumer := NewInboxConsumer(repo, "test-service")
	s.NoError(consumer.Wrap("h", func(context.Context, amqp.Delivery) error { return nil })(
		context.Background(), deliveryWithMessageID("msg_reclaim")))

	s.NotEqual("dead-pod", claimOwner)
	s.Equal(DefaultInboxLeaseSeconds, claimTTL)
}

// ---------------------------------------------------------------------------
// shutdown
// ---------------------------------------------------------------------------

// Shutdown cancels the delivery's context. If the outcome were written with it, the record would keep its lease for the full duration and be neither retryable nor visibly failed — the case where recording matters most.
func (s *InboxConsumerTestSuite) TestWrap_HandlerFails_RecordsFailureEvenWhenContextIsCancelled() {
	var markedFailed bool
	repo := &mockInboxRepo{
		tryInsertFn: func(context.Context, InboxRecordInput) (int64, error) { return 42, nil },
		markFailedFn: func(ctx context.Context, _ int64, _ string, _ string) error {
			s.NoError(ctx.Err(), "the outcome must not be written with the cancelled delivery context")
			markedFailed = true
			return nil
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	consumer := NewInboxConsumer(repo, "test-service")
	wrapped := consumer.Wrap("test-handler", func(context.Context, amqp.Delivery) error {
		cancel() // the pod goes away mid-handler
		return errors.New("interrupted")
	})

	err := wrapped(ctx, deliveryWithMessageID("msg_shutdown"))
	s.Error(err)
	s.True(markedFailed, "a lease left held by a cancelled write blocks the retry until it lapses")
}

// The same applies to a success: an attempt that finished must be recorded even if shutdown raced it.
func (s *InboxConsumerTestSuite) TestWrap_Success_CompletesEvenWhenContextIsCancelled() {
	var completed bool
	repo := &mockInboxRepo{
		tryInsertFn: func(context.Context, InboxRecordInput) (int64, error) { return 42, nil },
		completeFn: func(ctx context.Context, _ int64, _ string) (bool, error) {
			s.NoError(ctx.Err())
			completed = true
			return true, nil
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	consumer := NewInboxConsumer(repo, "test-service")
	wrapped := consumer.Wrap("test-handler", func(context.Context, amqp.Delivery) error {
		cancel()
		return nil
	})

	s.NoError(wrapped(ctx, deliveryWithMessageID("msg_shutdown_ok")))
	s.True(completed)
}

// A discard is a terminal record; it has to survive the same cancellation.
func (s *InboxConsumerTestSuite) TestWrap_Discard_RecordsEvenWhenContextIsCancelled() {
	var discarded bool
	repo := &mockInboxRepo{
		tryInsertFn: func(context.Context, InboxRecordInput) (int64, error) { return 42, nil },
		markDiscardedFn: func(ctx context.Context, _ int64, _ string, _ string) error {
			s.NoError(ctx.Err())
			discarded = true
			return nil
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	consumer := NewInboxConsumer(repo, "test-service")
	wrapped := consumer.Wrap("test-handler", func(hctx context.Context, _ amqp.Delivery) error {
		cancel()
		return consumer.Discard(hctx, "malformed")
	})

	s.NoError(wrapped(ctx, deliveryWithMessageID("msg_shutdown_discard")))
	s.True(discarded)
}

// ---------------------------------------------------------------------------
// discard edges
// ---------------------------------------------------------------------------

// A failure to write the terminal state must not turn a deliberate drop into a retry loop over a message that can never succeed.
func (s *InboxConsumerTestSuite) TestWrap_DiscardWriteFails_StillAcks() {
	repo := &mockInboxRepo{
		tryInsertFn: func(context.Context, InboxRecordInput) (int64, error) { return 42, nil },
		markDiscardedFn: func(context.Context, int64, string, string) error {
			return errors.New("db unavailable")
		},
		markFailedFn: func(context.Context, int64, string, string) error {
			s.Fail("a discard whose write failed is still not a retryable failure")
			return nil
		},
	}

	consumer := NewInboxConsumer(repo, "test-service")
	wrapped := consumer.Wrap("test-handler", func(ctx context.Context, _ amqp.Delivery) error {
		return consumer.Discard(ctx, "malformed")
	})

	s.NoError(wrapped(context.Background(), deliveryWithMessageID("msg_discard_fail")))
}

// Messages with no id skip dedup entirely, so a handler that discards one has no record to mark. It must ack rather than panic.
func (s *InboxConsumerTestSuite) TestDiscard_WithoutARecord_Acks() {
	repo := &mockInboxRepo{
		tryInsertFn: func(context.Context, InboxRecordInput) (int64, error) {
			s.Fail("TryInsert should not be called when there is no message ID")
			return 0, nil
		},
		markDiscardedFn: func(context.Context, int64, string, string) error {
			s.Fail("there is no record to discard")
			return nil
		},
	}

	consumer := NewInboxConsumer(repo, "test-service")
	wrapped := consumer.Wrap("test-handler", func(ctx context.Context, _ amqp.Delivery) error {
		return consumer.Discard(ctx, "malformed")
	})

	s.NoError(wrapped(context.Background(), deliveryWithoutMessageID()))
}

// A discarded message is not a failed one: it must not accumulate attempts or a last_error that would put it back in the retry population.
func (s *InboxConsumerTestSuite) TestWrap_Discard_DoesNotRecordAFailure() {
	repo := &mockInboxRepo{
		tryInsertFn: func(context.Context, InboxRecordInput) (int64, error) { return 42, nil },
		markFailedFn: func(context.Context, int64, string, string) error {
			s.Fail("a deliberate discard must not be recorded as a failed attempt")
			return nil
		},
		completeFn: func(context.Context, int64, string) (bool, error) {
			s.Fail("a discarded message was not processed")
			return false, nil
		},
	}

	consumer := NewInboxConsumer(repo, "test-service")
	wrapped := consumer.Wrap("test-handler", func(ctx context.Context, _ amqp.Delivery) error {
		return consumer.Discard(ctx, "no account on event")
	})

	s.NoError(wrapped(context.Background(), deliveryWithMessageID("msg_discard_only")))
}

// A handler that lost a completion race rolled its own work back; recording that as a failure would schedule a retry of work that is already done.
func (s *InboxConsumerTestSuite) TestWrap_AlreadyCompleted_DoesNotRecordAFailure() {
	repo := &mockInboxRepo{
		tryInsertFn: func(context.Context, InboxRecordInput) (int64, error) { return 5, nil },
		markFailedFn: func(context.Context, int64, string, string) error {
			s.Fail("losing a completion race is not a failure")
			return nil
		},
	}

	consumer := NewInboxConsumer(repo, "test-service")
	wrapped := consumer.Wrap("test-handler", func(context.Context, amqp.Delivery) error {
		return ErrInboxAlreadyCompleted
	})

	s.NoError(wrapped(context.Background(), deliveryWithMessageID("msg_lost_race")))
}

// ---------------------------------------------------------------------------
// lease ownership
// ---------------------------------------------------------------------------

// Every write that closes a record out is conditional on the lease, so each carries the owner that
// holds it. Without that, an attempt whose lease already lapsed can clear the lease of the consumer
// that claimed the record after it, and two handlers end up running the same message at once.

func (s *InboxConsumerTestSuite) TestWrap_CompletePassesTheLeaseOwner() {
	var completedWith string
	repo := &mockInboxRepo{
		tryInsertFn: func(context.Context, InboxRecordInput) (int64, error) { return 9, nil },
		completeFn: func(_ context.Context, _ int64, owner string) (bool, error) {
			completedWith = owner
			return true, nil
		},
	}

	consumer := NewInboxConsumer(repo, "test-service")
	var handlerSaw InboxLease
	wrapped := consumer.Wrap("test-handler", func(ctx context.Context, _ amqp.Delivery) error {
		handlerSaw, _ = InboxLeaseFromContext(ctx)
		return nil
	})

	s.NoError(wrapped(context.Background(), deliveryWithMessageID("msg_complete_owner")))
	s.NotEmpty(completedWith, "Complete is conditional on the lease, so it must say who holds it")
	s.Equal(handlerSaw.Owner, completedWith,
		"the owner the handler could commit with is the one the wrapper completes with")
}

func (s *InboxConsumerTestSuite) TestWrap_MarkFailedPassesTheLeaseOwner() {
	var failedWith string
	repo := &mockInboxRepo{
		tryInsertFn: func(context.Context, InboxRecordInput) (int64, error) { return 9, nil },
		markFailedFn: func(_ context.Context, _ int64, owner, _ string) error {
			failedWith = owner
			return nil
		},
	}

	consumer := NewInboxConsumer(repo, "test-service")
	wrapped := consumer.Wrap("test-handler", func(context.Context, amqp.Delivery) error {
		return errors.New("handler blew up")
	})

	s.Error(wrapped(context.Background(), deliveryWithMessageID("msg_failed_owner")))
	s.NotEmpty(failedWith,
		"releasing the lease on failure must name the holder, or it can release someone else's")
}

func (s *InboxConsumerTestSuite) TestDiscard_PassesTheLeaseOwner() {
	var discardedWith string
	repo := &mockInboxRepo{
		tryInsertFn: func(context.Context, InboxRecordInput) (int64, error) { return 9, nil },
		markDiscardedFn: func(_ context.Context, _ int64, owner, _ string) error {
			discardedWith = owner
			return nil
		},
	}

	consumer := NewInboxConsumer(repo, "test-service")
	wrapped := consumer.Wrap("test-handler", func(ctx context.Context, _ amqp.Delivery) error {
		return consumer.Discard(ctx, "malformed")
	})

	s.NoError(wrapped(context.Background(), deliveryWithMessageID("msg_discard_owner")))
	s.NotEmpty(discardedWith, "a terminal write is conditional on the lease too")
}

// ---------------------------------------------------------------------------
// ignored
// ---------------------------------------------------------------------------

// A message this handler was never going to act on is terminal like a discard, but it is not a
// failure. The distinction is what keeps the monitor's alerts about records that need a human:
// unsubscribed event types arrive constantly, and alerting on each one buries everything else.

func (s *InboxConsumerTestSuite) TestIgnore_RecordsIgnoredNotDiscarded() {
	var ignoredFor string
	repo := &mockInboxRepo{
		tryInsertFn: func(context.Context, InboxRecordInput) (int64, error) { return 11, nil },
		markIgnoredFn: func(_ context.Context, _ int64, _, reason string) error {
			ignoredFor = reason
			return nil
		},
		markDiscardedFn: func(context.Context, int64, string, string) error {
			s.Fail("a message that was never this handler's work is not a discard")
			return nil
		},
		markFailedFn: func(context.Context, int64, string, string) error {
			s.Fail("nor is it a failed attempt")
			return nil
		},
		completeFn: func(context.Context, int64, string) (bool, error) {
			s.Fail("and it must not be recorded as work that was applied")
			return false, nil
		},
	}

	consumer := NewInboxConsumer(repo, "test-service")
	wrapped := consumer.Wrap("test-handler", func(ctx context.Context, _ amqp.Delivery) error {
		return consumer.Ignore(ctx, "unhandled event type")
	})

	s.NoError(wrapped(context.Background(), deliveryWithMessageID("msg_ignored")),
		"the delivery is acked: there is nothing to retry")
	s.Equal("unhandled event type", ignoredFor)
}

// An ignored record is terminal, so a redelivery skips it rather than running the handler again.
func (s *InboxConsumerTestSuite) TestWrap_IgnoredRecordIsSkippedOnRedelivery() {
	repo := &mockInboxRepo{
		tryInsertFn: func(context.Context, InboxRecordInput) (int64, error) { return 0, mysqlDupError() },
		getByMessageAndHandler: func(context.Context, string, string) (*InboxRecord, error) {
			return &InboxRecord{ID: 11, Status: InboxStatusIgnored}, nil
		},
		claimFn: func(context.Context, int64, string, int) (bool, error) {
			s.Fail("a terminal record is never claimed")
			return false, nil
		},
	}

	consumer := NewInboxConsumer(repo, "test-service")
	wrapped := consumer.Wrap("test-handler", func(context.Context, amqp.Delivery) error {
		s.Fail("an ignored message must not be handled again")
		return nil
	})

	s.NoError(wrapped(context.Background(), deliveryWithMessageID("msg_ignored_again")))
}

// A delivery whose message id could not be resolved has no record to end. Ending it must not reach
// for a repository the consumer may not have — the old Discard returned early for exactly this case.
func (s *InboxConsumerTestSuite) TestIgnore_WithNoLeaseInContextDoesNotTouchTheRepo() {
	consumer := NewInboxConsumer(nil, "test-service")

	s.NotPanics(func() {
		s.NoError(consumer.Ignore(context.Background(), "no record"))
		s.NoError(consumer.Discard(context.Background(), "no record"))
	})
}
