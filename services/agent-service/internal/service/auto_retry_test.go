package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/open-mrp/api/services/agent-service/internal/domain"
	factorymock "github.com/open-mrp/api/services/agent-service/internal/domain/mock/factory"
	repositorymock "github.com/open-mrp/api/services/agent-service/internal/domain/mock/repository"
	agentdb "github.com/open-mrp/api/services/agent-service/internal/infrastructure/db"
	"github.com/open-mrp/api/services/agent-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/services/agent-service/internal/llm"
	"github.com/open-mrp/api/shared/contracts"
	"github.com/open-mrp/api/shared/messaging"
	"go.uber.org/mock/gomock"
)

// fakeOutboxRepo records the single outbox message a unit under test enqueues. The runner holds its
// OutboxRepo directly (not via the repo factory), so tests inject this rather than a gomock.
type fakeOutboxRepo struct {
	got       messaging.OutboxMessageInput
	created   bool
	createErr error
}

func (f *fakeOutboxRepo) Create(_ context.Context, input messaging.OutboxMessageInput) (int64, error) {
	f.got = input
	f.created = true
	return 1, f.createErr
}

func retryableErr() error {
	return &retryableRunError{cause: &llm.GatewayError{StatusCode: 503, Retryable: true}}
}

// maybeAutoRetry re-enqueues a transient, side-effect-free failure with backoff instead of failing the run.
func TestMaybeAutoRetry_ReenqueuesTransientFailure(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)

	runRepo := repositorymock.NewMockAgentRunRepo(ctrl)
	runRepo.EXPECT().MarkAutoRetrying(gomock.Any(), "agr_1").Return(int32(1), nil).Times(1)

	eventRepo := repositorymock.NewMockAgentRunEventRepo(ctrl)
	eventRepo.EXPECT().GetMaxSequence(gomock.Any(), "agr_1").Return(int32(4), nil).Times(1)
	eventRepo.EXPECT().Insert(gomock.Any(), gomock.Any()).Return(nil).Times(1)

	factory := factorymock.NewMockRepoFactory(ctrl)
	factory.EXPECT().NewAgentRunEventRepo().Return(eventRepo).AnyTimes()

	outbox := &fakeOutboxRepo{}
	s := &runnerSvc{repos: factory, outboxRepo: outbox} // no broker → emitEvent only inserts, never publishes

	run := &sqlc.AgentRun{ID: "agr_1", AccountID: "acct_1", RetryCount: 0, TriggerMessageID: agentdb.PgText("msg_9")}
	if !s.maybeAutoRetry(context.Background(), runRepo, run, retryableErr()) {
		t.Fatal("expected maybeAutoRetry to re-enqueue and return true")
	}

	if !outbox.created {
		t.Fatal("expected an outbox message to be created")
	}
	if outbox.got.MessageType != string(contracts.AgentCmdContinueRun) {
		t.Errorf("message type = %q, want %q", outbox.got.MessageType, contracts.AgentCmdContinueRun)
	}
	if outbox.got.DelaySeconds <= 0 {
		t.Errorf("expected a positive backoff delay, got %d", outbox.got.DelaySeconds)
	}
	var data messaging.AgentContinueRunData
	if err := json.Unmarshal(outbox.got.Payload.Data, &data); err != nil {
		t.Fatalf("payload not valid AgentContinueRunData: %v", err)
	}
	if data.AgentRunID != "agr_1" || data.AccountID != "acct_1" {
		t.Errorf("payload run/account = %q/%q", data.AgentRunID, data.AccountID)
	}
	if data.Message != "" {
		t.Errorf("resume message should be empty, got %q", data.Message)
	}
	if data.ReplyToMessageID != "msg_9" {
		t.Errorf("reply-to = %q, want msg_9", data.ReplyToMessageID)
	}
}

// A non-retryable error must fall through to terminal failure: no counter bump, no re-enqueue.
func TestMaybeAutoRetry_IgnoresNonRetryable(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	runRepo := repositorymock.NewMockAgentRunRepo(ctrl) // no calls expected
	outbox := &fakeOutboxRepo{}
	s := &runnerSvc{repos: factorymock.NewMockRepoFactory(ctrl), outboxRepo: outbox}

	run := &sqlc.AgentRun{ID: "agr_1", RetryCount: 0}
	if s.maybeAutoRetry(context.Background(), runRepo, run, errors.New("bad request")) {
		t.Error("non-retryable error must not auto-retry")
	}
	if outbox.created {
		t.Error("non-retryable error must not enqueue a resume")
	}
}

// Once the run has reached the auto-retry cap, a further transient failure becomes terminal.
func TestMaybeAutoRetry_StopsAtCap(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	runRepo := repositorymock.NewMockAgentRunRepo(ctrl) // no calls expected — cap checked before MarkAutoRetrying
	outbox := &fakeOutboxRepo{}
	s := &runnerSvc{repos: factorymock.NewMockRepoFactory(ctrl), outboxRepo: outbox}

	run := &sqlc.AgentRun{ID: "agr_1", RetryCount: domain.MaxAutoRetries}
	if s.maybeAutoRetry(context.Background(), runRepo, run, retryableErr()) {
		t.Error("at the auto-retry cap the failure must be terminal")
	}
	if outbox.created {
		t.Error("must not enqueue once at the cap")
	}
}

// If the resume can't be enqueued after the counter was bumped, the run is reverted to failed so it
// isn't left stuck in 'running' with no pending work (manual retry is still possible afterward).
func TestMaybeAutoRetry_RevertsOnEnqueueFailure(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)

	runRepo := repositorymock.NewMockAgentRunRepo(ctrl)
	runRepo.EXPECT().MarkAutoRetrying(gomock.Any(), "agr_1").Return(int32(1), nil).Times(1)
	runRepo.EXPECT().UpdateFailed(gomock.Any(), gomock.Any()).Return(nil).Times(1)

	outbox := &fakeOutboxRepo{createErr: errors.New("broker down")}
	s := &runnerSvc{repos: factorymock.NewMockRepoFactory(ctrl), outboxRepo: outbox}

	run := &sqlc.AgentRun{ID: "agr_1", AccountID: "acct_1", RetryCount: 0}
	if s.maybeAutoRetry(context.Background(), runRepo, run, retryableErr()) {
		t.Error("a failed enqueue must report not-retried so the caller fails the run")
	}
}
