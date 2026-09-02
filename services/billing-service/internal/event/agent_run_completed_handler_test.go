package event

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/open-mrp/api/services/billing-service/internal/domain"
	factorymock "github.com/open-mrp/api/services/billing-service/internal/domain/mock/factory"
	repositorymock "github.com/open-mrp/api/services/billing-service/internal/domain/mock/repository"
	"github.com/open-mrp/api/shared/contracts"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/messaging"
)

// The token accumulator is the one write in this service whose duplicate is additive rather than
// convergent: a second delivery does not land on the same total, it inflates it. These tests hold the
// line that the usage row and the marker saying the run was counted are one commit.

// ─── a transaction that can roll back ──────────────────────────────────────

type billingJournal struct {
	inTx        bool
	stagedRuns  int
	stagedMarks int
	runs        int
	marks       int
}

func (j *billingJournal) begin() { j.inTx = true; j.stagedRuns, j.stagedMarks = 0, 0 }

func (j *billingJournal) rollback() { j.inTx = false; j.stagedRuns, j.stagedMarks = 0, 0 }

func (j *billingJournal) commit() {
	j.inTx = false
	j.runs += j.stagedRuns
	j.marks += j.stagedMarks
	j.stagedRuns, j.stagedMarks = 0, 0
}

type billingTxManager struct {
	factory   domain.RepoFactory
	journal   *billingJournal
	calls     int
	commitErr *apierror.APIError
}

func (m *billingTxManager) WithTx(ctx context.Context, fn func(context.Context, domain.RepoFactory) *apierror.APIError) *apierror.APIError {
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

func (m *billingTxManager) WithTxSavepoint(ctx context.Context, fn func(context.Context, domain.RepoFactory, db.SavepointRunner) *apierror.APIError) *apierror.APIError {
	return m.WithTx(ctx, func(ctx context.Context, f domain.RepoFactory) *apierror.APIError {
		return fn(ctx, f, billingNoopSavepoint{})
	})
}

type billingNoopSavepoint struct{}

func (billingNoopSavepoint) Run(ctx context.Context, fn func(context.Context) *apierror.APIError) *apierror.APIError {
	return fn(ctx)
}

// ─── inboxes ───────────────────────────────────────────────────────────────

type billingTxInbox struct {
	journal         *billingJournal
	completeReturns bool
	completeErr     error
	completeCalls   int
}

func (r *billingTxInbox) TryInsert(context.Context, messaging.InboxRecordInput) (int64, error) {
	return 0, errors.New("not used inside a transaction")
}

func (r *billingTxInbox) GetByMessageAndHandler(context.Context, string, string) (*messaging.InboxRecord, error) {
	return nil, errors.New("not used inside a transaction")
}

func (r *billingTxInbox) Claim(context.Context, int64, string, int) (bool, error) {
	return false, errors.New("not used inside a transaction")
}

func (r *billingTxInbox) Complete(context.Context, int64) (bool, error) {
	r.completeCalls++
	if r.completeErr != nil {
		return false, r.completeErr
	}
	if !r.completeReturns {
		return false, nil
	}
	r.journal.stagedMarks++
	return true, nil
}

func (r *billingTxInbox) MarkFailed(context.Context, int64, string) error { return nil }

func (r *billingTxInbox) MarkDiscarded(context.Context, int64, string) error { return nil }

type billingOuterInbox struct {
	record    *messaging.InboxRecord
	insertErr error
	failed    int
	completed int
}

func (r *billingOuterInbox) TryInsert(context.Context, messaging.InboxRecordInput) (int64, error) {
	if r.insertErr != nil {
		return 0, r.insertErr
	}
	return 7, nil
}

func (r *billingOuterInbox) GetByMessageAndHandler(context.Context, string, string) (*messaging.InboxRecord, error) {
	if r.record == nil {
		return nil, errors.New("not found")
	}
	return r.record, nil
}

func (r *billingOuterInbox) Claim(context.Context, int64, string, int) (bool, error) {
	return true, nil
}

func (r *billingOuterInbox) Complete(context.Context, int64) (bool, error) {
	r.completed++
	return true, nil
}

func (r *billingOuterInbox) MarkFailed(context.Context, int64, string) error { r.failed++; return nil }

func (r *billingOuterInbox) MarkDiscarded(context.Context, int64, string) error { return nil }

// ─── suite ─────────────────────────────────────────────────────────────────

type AgentTokenBillingTestSuite struct {
	suite.Suite
	ctrl *gomock.Controller

	journal    *billingJournal
	txManager  *billingTxManager
	txInbox    *billingTxInbox
	outerInbox *billingOuterInbox
	handler    *AgentTokenBillingHandler
	tokenRepo  *repositorymock.MockAgentTokenBillingRepo
}

func (s *AgentTokenBillingTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.journal = &billingJournal{}
	s.txInbox = &billingTxInbox{journal: s.journal, completeReturns: true}
	s.outerInbox = &billingOuterInbox{}

	periodEnd := time.Now().AddDate(0, 1, 0)
	usageRepo := repositorymock.NewMockAccountUsageRepo(s.ctrl)
	usageRepo.EXPECT().GetAccountSubscriptionInfo(gomock.Any(), gomock.Any()).
		Return(&domain.AccountSubscriptionInfo{SubscriptionCurrentPeriodEnd: &periodEnd}, nil).AnyTimes()

	s.tokenRepo = repositorymock.NewMockAgentTokenBillingRepo(s.ctrl)
	s.tokenRepo.EXPECT().UpsertAgentTokenBilling(gomock.Any(), gomock.Any()).
		DoAndReturn(func(context.Context, domain.UpsertAgentTokenBillingParams) *apierror.APIError {
			s.journal.stagedRuns++
			return nil
		}).AnyTimes()

	factory := factorymock.NewMockRepoFactory(s.ctrl)
	factory.EXPECT().NewAccountUsageRepo().Return(usageRepo).AnyTimes()
	factory.EXPECT().NewAgentTokenBillingRepo().Return(s.tokenRepo).AnyTimes()
	factory.EXPECT().NewInboxRepo().Return(s.txInbox).AnyTimes()

	s.txManager = &billingTxManager{factory: factory, journal: s.journal}
	s.handler = NewAgentTokenBillingHandler(s.tokenRepo, factory, s.txManager)
}

func (s *AgentTokenBillingTestSuite) TearDownTest() { s.ctrl.Finish() }

func TestAgentTokenBillingTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(AgentTokenBillingTestSuite))
}

func (s *AgentTokenBillingTestSuite) deliver(messageID string, tokens int) error {
	data, err := json.Marshal(messaging.AgentRunCompletedData{
		AgentRunID:       "arun_1",
		AccountID:        "acct_1",
		BillingAccountID: "acct_1",
		InputTokens:      tokens / 2,
		OutputTokens:     tokens / 2,
		TotalTokens:      tokens,
	})
	s.Require().NoError(err)
	body, err := json.Marshal(contracts.AmqpMessage{MessageID: messageID, Data: data})
	s.Require().NoError(err)

	inbox := messaging.NewInboxConsumer(s.outerInbox, "billing-service")
	return inbox.Wrap("billing.agent_run_completed", s.handler.Handle)(
		context.Background(), amqp.Delivery{MessageId: messageID, Body: body})
}

func (s *AgentTokenBillingTestSuite) TestSuccess_UsageAndMarkerCommitTogether() {
	s.NoError(s.deliver("msg_ok", 1000))

	s.Equal(1, s.journal.runs)
	s.Equal(1, s.journal.marks, "the marker must commit with the usage row it accounts for")
}

// The window this closed: a committed usage row whose marker was lost is counted again on redelivery,
// and because the upsert accumulates, the account's totals grow every time.
func (s *AgentTokenBillingTestSuite) TestCommitFails_UsageDoesNotSurvive() {
	s.txManager.commitErr = apierror.NewInternalError(errors.New("deadlock"), "commit failed")

	s.Error(s.deliver("msg_commit_fail", 1000))

	s.Zero(s.journal.runs, "a run that was not marked must not have been counted")
	s.Zero(s.journal.marks)
	s.Equal(1, s.outerInbox.failed)
}

func (s *AgentTokenBillingTestSuite) TestConcurrentAttemptWon_UsageIsRolledBack() {
	s.txInbox.completeReturns = false

	s.NoError(s.deliver("msg_raced", 1000), "losing the race is not a failure to report")

	s.Zero(s.journal.runs, "the loser must not add its tokens on top of the winner's")
	s.Equal(1, s.txInbox.completeCalls)
	s.Zero(s.outerInbox.failed)
}

func (s *AgentTokenBillingTestSuite) TestRedelivery_AlreadyProcessed_CountsNothing() {
	s.outerInbox.insertErr = &mysql.MySQLError{Number: 1062, Message: "Duplicate entry"}
	s.outerInbox.record = &messaging.InboxRecord{ID: 7, Status: messaging.InboxStatusProcessed}

	s.NoError(s.deliver("msg_dup", 1000))

	s.Zero(s.journal.runs)
	s.Zero(s.txManager.calls, "no transaction should even be opened")
}

// A redelivery arriving while the first attempt is still working must not add a second count.
func (s *AgentTokenBillingTestSuite) TestRedelivery_LeaseHeld_CountsNothing() {
	live := time.Now().Add(time.Minute)
	s.outerInbox.insertErr = &mysql.MySQLError{Number: 1062, Message: "Duplicate entry"}
	s.outerInbox.record = &messaging.InboxRecord{
		ID: 7, Status: messaging.InboxStatusReceived, LockExpiresAt: &live,
	}

	err := s.deliver("msg_inflight", 1000)

	s.ErrorIs(err, messaging.ErrInboxLeaseHeld)
	s.Zero(s.journal.runs)
	s.Zero(s.txManager.calls)
}
