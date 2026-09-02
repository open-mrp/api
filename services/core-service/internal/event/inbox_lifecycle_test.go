package event

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	factorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/factory"
	repositorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/repository"
	"github.com/open-mrp/api/shared/contracts"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/messaging"
	"github.com/open-mrp/api/shared/tracing"
)

// These tests are about the delivery lifecycle rather than the inventory arithmetic, which
// BatchScannedConsumerTestSuite already covers. What is asserted here is the property the whole
// design rests on: the ledger movements and the inbox marker that says they happened are one
// commit. Anything that can split them — a failure part-way, a concurrent delivery, a lost commit —
// must leave both sides unwritten, because a committed movement with no marker is what a redelivery
// or a replay applies a second time.

// ─── a transaction that can actually roll back ─────────────────────────────

type effectKind string

const (
	effectInventory     effectKind = "inventory"
	effectInboxComplete effectKind = "inbox_complete"
)

type effect struct {
	kind    effectKind
	itemID  string
	measure decimal.Decimal
}

// txJournal separates writes that a transaction has staged from writes that survived its commit, so
// a test can ask what is actually in the database after a failure rather than what was attempted.
type txJournal struct {
	inTx      bool
	staged    []effect
	committed []effect
}

func (j *txJournal) record(e effect) {
	if j.inTx {
		j.staged = append(j.staged, e)
		return
	}
	j.committed = append(j.committed, e)
}

func (j *txJournal) begin()    { j.inTx = true; j.staged = nil }
func (j *txJournal) rollback() { j.inTx = false; j.staged = nil }
func (j *txJournal) commit() {
	j.inTx = false
	j.committed = append(j.committed, j.staged...)
	j.staged = nil
}

func (j *txJournal) committedOf(kind effectKind) []effect {
	var out []effect
	for _, e := range j.committed {
		if e.kind == kind {
			out = append(out, e)
		}
	}
	return out
}

// fakeTxManager runs the callback against the journal and honors the rollback contract. commitErr
// simulates the transaction failing at commit — after the callback returned cleanly — which is the
// case that decides whether an unmarked message is safe to redeliver.
type fakeTxManager struct {
	factory   domain.RepoFactory
	journal   *txJournal
	calls     int
	commitErr *apierror.APIError
}

func (m *fakeTxManager) WithTx(ctx context.Context, fn func(context.Context, domain.RepoFactory) *apierror.APIError) *apierror.APIError {
	m.calls++
	m.journal.begin()
	if apiErr := fn(ctx, m.factory); apiErr != nil {
		m.journal.rollback()
		return apiErr
	}
	if m.commitErr != nil {
		m.journal.rollback()
		return m.commitErr
	}
	m.journal.commit()
	return nil
}

func (m *fakeTxManager) WithTxSavepoint(ctx context.Context, fn func(context.Context, domain.RepoFactory, db.SavepointRunner) *apierror.APIError) *apierror.APIError {
	return m.WithTx(ctx, func(ctx context.Context, f domain.RepoFactory) *apierror.APIError {
		return fn(ctx, f, noopSavepoint{})
	})
}

type noopSavepoint struct{}

func (noopSavepoint) Run(ctx context.Context, fn func(context.Context) *apierror.APIError) *apierror.APIError {
	return fn(ctx)
}

// ─── an inbox that records into the same journal ───────────────────────────

// journalInboxRepo is the transaction-scoped inbox. Complete records into the journal rather than
// writing immediately, which is the whole point: if the transaction rolls back, the marker goes with
// the work.
type journalInboxRepo struct {
	journal *txJournal
	// completeReturns is what the conditional UPDATE matched. false means a concurrent attempt
	// completed the message first.
	completeReturns bool
	completeErr     error
	completeCalls   int
}

func (r *journalInboxRepo) TryInsert(context.Context, messaging.InboxRecordInput) (int64, error) {
	return 0, errors.New("not used inside a transaction")
}

func (r *journalInboxRepo) GetByMessageAndHandler(context.Context, string, string) (*messaging.InboxRecord, error) {
	return nil, errors.New("not used inside a transaction")
}

func (r *journalInboxRepo) Claim(context.Context, int64, string, int) (bool, error) {
	return false, errors.New("not used inside a transaction")
}

func (r *journalInboxRepo) Complete(_ context.Context, id int64) (bool, error) {
	r.completeCalls++
	if r.completeErr != nil {
		return false, r.completeErr
	}
	if !r.completeReturns {
		return false, nil
	}
	r.journal.record(effect{kind: effectInboxComplete, measure: decimal.NewFromInt(id)})
	return true, nil
}

func (r *journalInboxRepo) MarkFailed(context.Context, int64, string) error { return nil }

func (r *journalInboxRepo) MarkDiscarded(context.Context, int64, string) error { return nil }

// outerInboxRepo is the pool-scoped inbox the wrapper itself uses, outside any transaction.
type outerInboxRepo struct {
	record       *messaging.InboxRecord
	inserted     bool
	completed    int
	failed       int
	discarded    int
	discardedFor []string
	insertErr    error
}

func (r *outerInboxRepo) TryInsert(_ context.Context, _ messaging.InboxRecordInput) (int64, error) {
	if r.insertErr != nil {
		return 0, r.insertErr
	}
	r.inserted = true
	return 42, nil
}

func (r *outerInboxRepo) GetByMessageAndHandler(context.Context, string, string) (*messaging.InboxRecord, error) {
	if r.record == nil {
		return nil, errors.New("not found")
	}
	return r.record, nil
}

func (r *outerInboxRepo) Claim(context.Context, int64, string, int) (bool, error) { return true, nil }

func (r *outerInboxRepo) Complete(context.Context, int64) (bool, error) {
	r.completed++
	return true, nil
}

func (r *outerInboxRepo) MarkFailed(context.Context, int64, string) error {
	r.failed++
	return nil
}

func (r *outerInboxRepo) MarkDiscarded(_ context.Context, _ int64, reason string) error {
	r.discarded++
	r.discardedFor = append(r.discardedFor, reason)
	return nil
}

// ─── suite ─────────────────────────────────────────────────────────────────

type InboxLifecycleTestSuite struct {
	suite.Suite
	ctrl *gomock.Controller

	journal    *txJournal
	txManager  *fakeTxManager
	txInbox    *journalInboxRepo
	outerInbox *outerInboxRepo
	consumer   *BatchScannedConsumer
	stepRepo   *repositorymock.MockProductionStepQueryRepo
}

func (s *InboxLifecycleTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.journal = &txJournal{}
	s.txInbox = &journalInboxRepo{journal: s.journal, completeReturns: true}
	s.outerInbox = &outerInboxRepo{}

	s.stepRepo = repositorymock.NewMockProductionStepQueryRepo(s.ctrl)
	s.stepRepo.EXPECT().Find(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(step("1", unitPair), nil).AnyTimes()

	unitConv := repositorymock.NewMockUnitConversionRepo(s.ctrl)
	unitConv.EXPECT().ConvertValue(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, d decimal.Decimal, _, _ string) (decimal.Decimal, *apierror.APIError) {
			return d, nil
		}).AnyTimes()
	unitConv.EXPECT().GetUnitFactors(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(map[string]domain.UnitFactors{}, nil).AnyTimes()

	mutation := repositorymock.NewMockInventoryMutationRepo(s.ctrl)
	mutation.EXPECT().UpdateInventory(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, p domain.InventoryUpdateParams) *apierror.APIError {
			s.journal.record(effect{kind: effectInventory, itemID: p.ItemID, measure: p.Measure})
			return nil
		}).AnyTimes()
	mutation.EXPECT().CreateInventoryLog(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	mutation.EXPECT().CreateInventoryChangeLog(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	reservation := repositorymock.NewMockInventoryReservationRepo(s.ctrl)
	reservation.EXPECT().LockItemForLedger(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	batchRepo := repositorymock.NewMockBatchRepo(s.ctrl)
	batchRepo.EXPECT().FindLineageShortfall(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()

	itemRepo := repositorymock.NewMockItemRepo(s.ctrl)
	itemRepo.EXPECT().Get(gomock.Any(), gomock.Any()).
		Return(nil, apierror.NewResourceNotFoundError("item")).AnyTimes()

	inventoryQuery := repositorymock.NewMockInventoryQueryRepo(s.ctrl)
	inventoryQuery.EXPECT().FetchPhysicalInventory(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(decimal.Zero, nil).AnyTimes()
	inventoryQuery.EXPECT().FetchPhysicalInventoryBaseForItems(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(map[string]decimal.Decimal{}, nil).AnyTimes()

	factory := factorymock.NewMockRepoFactory(s.ctrl)
	factory.EXPECT().NewProductionStepQueryRepo().Return(s.stepRepo).AnyTimes()
	factory.EXPECT().NewUnitConversionRepo().Return(unitConv).AnyTimes()
	factory.EXPECT().NewInventoryMutationRepo().Return(mutation).AnyTimes()
	factory.EXPECT().NewInventoryReservationRepo().Return(reservation).AnyTimes()
	factory.EXPECT().NewBatchRepo().Return(batchRepo).AnyTimes()
	factory.EXPECT().NewOrderQueryRepo().Return(repositorymock.NewMockOrderQueryRepo(s.ctrl)).AnyTimes()
	factory.EXPECT().NewMaterialDemandRepo().Return(repositorymock.NewMockMaterialDemandRepo(s.ctrl)).AnyTimes()
	factory.EXPECT().NewItemRepo().Return(itemRepo).AnyTimes()
	factory.EXPECT().NewInventoryQueryRepo().Return(inventoryQuery).AnyTimes()
	factory.EXPECT().NewOutboxRepo().Return(recordingOutboxRepo{}).AnyTimes()
	factory.EXPECT().NewInboxRepo().Return(s.txInbox).AnyTimes()

	s.txManager = &fakeTxManager{factory: factory, journal: s.journal}
	s.consumer = &BatchScannedConsumer{
		inboxConsumer: messaging.NewInboxConsumer(s.outerInbox, "core-service"),
		repos:         factory,
		txManager:     s.txManager,
		tracer:        tracing.GetTracer("test.inbox_lifecycle"),
	}
}

func (s *InboxLifecycleTestSuite) TearDownTest() { s.ctrl.Finish() }

func TestInboxLifecycleTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(InboxLifecycleTestSuite))
}

// deliver drives the event through the same inbox wrapper the live consumer registers, so the test
// exercises dedup, lease and recovery point together rather than the handler in isolation.
func (s *InboxLifecycleTestSuite) deliver(evt domain.BatchScannedEvent, messageID string) error {
	data, err := json.Marshal(evt)
	s.Require().NoError(err)
	body, err := json.Marshal(contracts.AmqpMessage{MessageID: messageID, Data: data})
	s.Require().NoError(err)

	wrapped := s.consumer.inboxConsumer.Wrap("core.batch_scanned_inventory", s.consumer.handleMessage)
	return wrapped(context.Background(), amqp.Delivery{MessageId: messageID, Body: body})
}

// ─── the atomicity property ────────────────────────────────────────────────

func (s *InboxLifecycleTestSuite) TestSuccess_MovementsAndMarkerCommitTogether() {
	s.NoError(s.deliver(scanEvent("10", unitPair), "msg_ok"))

	s.NotEmpty(s.journal.committedOf(effectInventory), "the scan should have moved inventory")
	s.Len(s.journal.committedOf(effectInboxComplete), 1,
		"the marker must commit with the movements, not after them")
}

// A failure part-way must leave nothing: no movements, and no marker claiming there were any.
func (s *InboxLifecycleTestSuite) TestHandlerFails_NothingCommits() {
	s.stepRepo = repositorymock.NewMockProductionStepQueryRepo(s.ctrl)
	boom := apierror.NewInternalError(errors.New("boom"), "step lookup failed")

	// Fail on the second lookup — the one inside the transaction — so the handler gets far enough to
	// have staged work before it dies.
	gomock.InOrder(
		s.stepRepo.EXPECT().Find(gomock.Any(), gomock.Any(), gomock.Any()).Return(step("1", unitPair), nil),
		s.stepRepo.EXPECT().Find(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, boom),
	)
	s.rebindStepRepo()

	err := s.deliver(scanEvent("10", unitPair), "msg_fail")

	s.Error(err)
	s.Empty(s.journal.committed, "a failed attempt must leave the ledger untouched")
	s.Equal(1, s.outerInbox.failed, "the failure has to be recorded so the message can be retried")
	s.Zero(s.outerInbox.discarded)
}

// The case the design exists for: the callback succeeded but the commit did not. Neither the
// movements nor the marker may survive, so the redelivery finds a message that genuinely never ran.
func (s *InboxLifecycleTestSuite) TestCommitFails_NeitherMovementsNorMarkerSurvive() {
	s.txManager.commitErr = apierror.NewInternalError(errors.New("lock wait timeout"), "commit failed")

	err := s.deliver(scanEvent("10", unitPair), "msg_commit_fail")

	s.Error(err)
	s.Empty(s.journal.committedOf(effectInventory))
	s.Empty(s.journal.committedOf(effectInboxComplete))
	s.Equal(1, s.outerInbox.failed)
}

// A concurrent delivery that lost the race must discard its own work rather than apply it on top of
// the winner's. Nothing failed, so the delivery is acked rather than retried.
func (s *InboxLifecycleTestSuite) TestConcurrentAttemptWon_WorkIsRolledBackAndAcked() {
	s.txInbox.completeReturns = false

	err := s.deliver(scanEvent("10", unitPair), "msg_raced")

	s.NoError(err, "losing the race is not a failure to report")
	s.Empty(s.journal.committed, "the loser's movements must not reach the ledger")
	s.Equal(1, s.txInbox.completeCalls)
	s.Zero(s.outerInbox.failed, "a rolled-back duplicate must not be queued for retry")
}

// The marker write failing is a failure like any other: the work rolls back with it.
func (s *InboxLifecycleTestSuite) TestMarkerWriteFails_WorkIsRolledBack() {
	s.txInbox.completeErr = errors.New("db unavailable")

	err := s.deliver(scanEvent("10", unitPair), "msg_marker_fail")

	s.Error(err)
	s.Empty(s.journal.committed)
	s.Equal(1, s.outerInbox.failed)
}

// ─── redelivery ────────────────────────────────────────────────────────────

// A message already applied must not be applied again, however it arrives.
func (s *InboxLifecycleTestSuite) TestRedelivery_AlreadyProcessed_MovesNothing() {
	s.outerInbox.insertErr = duplicateEntry()
	s.outerInbox.record = &messaging.InboxRecord{ID: 42, Status: messaging.InboxStatusProcessed}

	s.NoError(s.deliver(scanEvent("10", unitPair), "msg_dup"))
	s.Empty(s.journal.committed)
	s.Zero(s.txManager.calls, "no transaction should even be opened")
}

// A redelivery landing while the first attempt still holds the lease must not run alongside it.
func (s *InboxLifecycleTestSuite) TestRedelivery_LeaseHeld_MovesNothing() {
	live := nowPlus(60)
	s.outerInbox.insertErr = duplicateEntry()
	s.outerInbox.record = &messaging.InboxRecord{
		ID: 42, Status: messaging.InboxStatusReceived, LockExpiresAt: &live,
	}

	err := s.deliver(scanEvent("10", unitPair), "msg_inflight")

	s.ErrorIs(err, messaging.ErrInboxLeaseHeld)
	s.Empty(s.journal.committed)
	s.Zero(s.txManager.calls)
}

// A discarded message is terminal; redelivering it must not resurrect the work.
func (s *InboxLifecycleTestSuite) TestRedelivery_Discarded_MovesNothing() {
	s.outerInbox.insertErr = duplicateEntry()
	s.outerInbox.record = &messaging.InboxRecord{ID: 42, Status: messaging.InboxStatusDiscarded}

	s.NoError(s.deliver(scanEvent("10", unitPair), "msg_discarded"))
	s.Empty(s.journal.committed)
	s.Zero(s.txManager.calls)
}

// ─── messages that can never be applied ────────────────────────────────────

// A malformed event is recorded as discarded rather than acked as processed, which is what made a
// dropped scan indistinguishable from an applied one.
func (s *InboxLifecycleTestSuite) TestMalformedEvent_IsDiscardedNotProcessed() {
	evt := scanEvent("10", unitPair)
	evt.AccountID = ""

	s.NoError(s.deliver(evt, "msg_no_account"))

	s.Equal(1, s.outerInbox.discarded)
	s.Contains(s.outerInbox.discardedFor[0], "account")
	s.Zero(s.outerInbox.completed, "a dropped message must not be recorded as processed")
	s.Empty(s.journal.committed)
}

func (s *InboxLifecycleTestSuite) TestMissingProductionStep_IsDiscarded() {
	evt := scanEvent("10", unitPair)
	evt.ProductionStepID = ""

	s.NoError(s.deliver(evt, "msg_no_step"))

	s.Equal(1, s.outerInbox.discarded)
	s.Zero(s.txManager.calls)
}

func (s *InboxLifecycleTestSuite) TestMissingBatch_IsDiscarded() {
	evt := scanEvent("10", unitPair)
	evt.BatchID = ""

	s.NoError(s.deliver(evt, "msg_no_batch"))

	s.Equal(1, s.outerInbox.discarded)
	s.Zero(s.txManager.calls)
}

// The routing changed under the batch. This can never succeed on retry, so it ends terminally rather
// than burning the backoff ladder — and it must not commit an empty transaction that records success.
func (s *InboxLifecycleTestSuite) TestStepNoLongerProducesScannedItem_IsDiscardedWithNothingApplied() {
	evt := scanEvent("10", unitPair)
	evt.ItemID = "it_something_else"

	s.NoError(s.deliver(evt, "msg_item_mismatch"))

	s.Equal(1, s.outerInbox.discarded)
	s.Empty(s.journal.committed, "a discarded scan must not leave a marker or a movement behind")
	s.Zero(s.outerInbox.completed)
}

// ─── helpers ───────────────────────────────────────────────────────────────

func (s *InboxLifecycleTestSuite) rebindStepRepo() {
	factory := factorymock.NewMockRepoFactory(s.ctrl)
	old := s.txManager.factory
	factory.EXPECT().NewProductionStepQueryRepo().Return(s.stepRepo).AnyTimes()
	factory.EXPECT().NewUnitConversionRepo().DoAndReturn(old.NewUnitConversionRepo).AnyTimes()
	factory.EXPECT().NewInventoryMutationRepo().DoAndReturn(old.NewInventoryMutationRepo).AnyTimes()
	factory.EXPECT().NewInventoryReservationRepo().DoAndReturn(old.NewInventoryReservationRepo).AnyTimes()
	factory.EXPECT().NewBatchRepo().DoAndReturn(old.NewBatchRepo).AnyTimes()
	factory.EXPECT().NewOrderQueryRepo().DoAndReturn(old.NewOrderQueryRepo).AnyTimes()
	factory.EXPECT().NewMaterialDemandRepo().DoAndReturn(old.NewMaterialDemandRepo).AnyTimes()
	factory.EXPECT().NewItemRepo().DoAndReturn(old.NewItemRepo).AnyTimes()
	factory.EXPECT().NewInventoryQueryRepo().DoAndReturn(old.NewInventoryQueryRepo).AnyTimes()
	factory.EXPECT().NewOutboxRepo().DoAndReturn(old.NewOutboxRepo).AnyTimes()
	factory.EXPECT().NewInboxRepo().Return(s.txInbox).AnyTimes()

	s.txManager.factory = factory
	s.consumer.repos = factory
}

// duplicateEntry is the driver error a second insert of the same (message_id, handler) raises; only
// the real type routes Wrap into its dedup path.
func duplicateEntry() error { return &mysql.MySQLError{Number: 1062, Message: "Duplicate entry"} }

func nowPlus(seconds int) time.Time { return time.Now().Add(time.Duration(seconds) * time.Second) }
