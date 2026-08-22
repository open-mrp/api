package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/open-mrp/api/services/agent-service/internal/domain"
	factorymock "github.com/open-mrp/api/services/agent-service/internal/domain/mock/factory"
	repositorymock "github.com/open-mrp/api/services/agent-service/internal/domain/mock/repository"
	agentdb "github.com/open-mrp/api/services/agent-service/internal/infrastructure/db"
	"github.com/open-mrp/api/services/agent-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/contracts"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/messaging"
	"go.uber.org/mock/gomock"
)

// fakeTxManager runs the callback inline with a fixed factory, so service methods that wrap work in
// withTx can be unit-tested without a real database.
type fakeTxManager struct{ f domain.RepoFactory }

func (m fakeTxManager) WithTx(ctx context.Context, fn func(context.Context, domain.RepoFactory) *apierror.APIError) *apierror.APIError {
	return fn(ctx, m.f)
}

// fakeOutbox captures the outbox messages a service method enqueues.
type fakeOutbox struct {
	inputs []messaging.OutboxMessageInput
}

func (f *fakeOutbox) Create(_ context.Context, in messaging.OutboxMessageInput) (int64, error) {
	f.inputs = append(f.inputs, in)
	return int64(len(f.inputs)), nil
}

// continueChatRun decides whether a conversation reply resumes the replied-to run or falls through to a
// fresh one. These cases cover the bail-outs that return false (caller then starts a clean run seeded
// with conversation history) — a missing/foreign run, a diverged run (poisoned by a private console
// turn), and an in-flight run (running/pending: nothing to resume yet, not terminal to inherit). Terminal
// runs — failed/cancelled/completed — do NOT fall through; they fork an heir; see TestForkDeadChatRun.
func TestContinueChatRun_FallsThrough(t *testing.T) {
	t.Parallel()

	const acct = "acc_1"
	cases := []struct {
		name   string
		run    *sqlc.AgentRun
		runErr *apierror.APIError
	}{
		{
			name:   "missing run starts fresh",
			runErr: apierror.NewResourceNotFoundError("nope"),
		},
		{
			name: "wrong account starts fresh",
			run:  &sqlc.AgentRun{ID: "agr_1", AccountID: "other", StatusCode: domain.RunStatusAwaitingInput},
		},
		{
			// A run poisoned by a private agent-run-console turn must never be resumed or inherited,
			// regardless of status — it would leak the off-conversation fork into the reply.
			name: "diverged run starts fresh",
			run:  &sqlc.AgentRun{ID: "agr_1", AccountID: acct, StatusCode: domain.RunStatusAwaitingInput, DivergedFromConversation: true},
		},
		{
			name: "running run starts fresh",
			run:  &sqlc.AgentRun{ID: "agr_1", AccountID: acct, StatusCode: domain.RunStatusRunning},
		},
		{
			name: "pending run starts fresh",
			run:  &sqlc.AgentRun{ID: "agr_1", AccountID: acct, StatusCode: domain.RunStatusPending},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			runRepo := repositorymock.NewMockAgentRunRepo(ctrl)
			runRepo.EXPECT().GetByID(gomock.Any(), "agr_1").Return(tc.run, tc.runErr).Times(1)
			factory := factorymock.NewMockRepoFactory(ctrl)
			factory.EXPECT().NewAgentRunRepo().Return(runRepo).Times(1)

			s := &agentDefSvcImpl{repos: factory}
			continued, apiErr := s.continueChatRun(context.Background(), domain.ChatRunInput{
				AccountID:     acct,
				ContinueRunID: "agr_1",
				Message:       "hello again",
			})
			if apiErr != nil {
				t.Fatalf("continueChatRun returned error: %v", apiErr)
			}
			if continued {
				t.Errorf("continued = true, want false (should fall through to a fresh run)")
			}
		})
	}
}

// A reply to a terminal chat run (failed, cancelled, or completed) forks an heir run that inherits the
// dead run's transcript minus the terminal failure markers ("the failed responses"), then drives the
// reply through it. This verifies the heir is created from the dead run's identity, the transcript is
// copied/re-sequenced with error+cancelled events dropped and action links cleared, and the reply is
// enqueued as the heir's turn.
func TestForkDeadChatRun(t *testing.T) {
	t.Parallel()

	const (
		acct      = "acc_1"
		deadID    = "agr_dead"
		convID    = "cnv_1"
		defID     = "agd_1"
		configID  = "agc_1"
		triggerID = "msg_reply"
		reply     = "actually, also do X"
	)

	for _, deadStatus := range []string{domain.RunStatusFailed, domain.RunStatusCancelled, domain.RunStatusCompleted} {
		t.Run(deadStatus, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)

			dead := &sqlc.AgentRun{
				ID:                deadID,
				AccountID:         acct,
				AgentDefinitionID: defID,
				AgentConfigID:     agentdb.PgText(configID),
				ConversationID:    agentdb.PgText(convID),
				StatusCode:        deadStatus,
			}
			// Transcript ending in the failure marker; the marker (and a cancelled one) must be dropped,
			// while the real turns — including a tool call linked to an action — are inherited.
			events := []sqlc.AgentRunEvent{
				{StepType: "trigger_received", Title: "Run triggered", Sequence: 0},
				{StepType: "user_message", Title: "User message", Content: agentdb.PgText("do the thing"), Sequence: 1},
				{StepType: "tool_call", Title: "search", Sequence: 2, AgentActionID: agentdb.PgText("aac_1"), Metadata: json.RawMessage(`{"tool_use_id":"tu_1"}`)},
				{StepType: "tool_result", Title: "search result", Sequence: 3, Metadata: json.RawMessage(`{"tool_use_id":"tu_1"}`)},
				{StepType: stepTypeCancelled, Title: "Run cancelled", Sequence: 4},
				{StepType: stepTypeError, Title: "Run failed", Sequence: 5},
			}
			const wantCopied = 4

			runRepo := repositorymock.NewMockAgentRunRepo(ctrl)
			eventRepo := repositorymock.NewMockAgentRunEventRepo(ctrl)
			outbox := &fakeOutbox{}
			factory := factorymock.NewMockRepoFactory(ctrl)
			factory.EXPECT().NewAgentRunRepo().Return(runRepo).AnyTimes()
			factory.EXPECT().NewAgentRunEventRepo().Return(eventRepo).AnyTimes()
			factory.EXPECT().NewOutboxRepo().Return(outbox).AnyTimes()

			runRepo.EXPECT().GetByID(gomock.Any(), deadID).Return(dead, nil).Times(1)
			eventRepo.EXPECT().ListByRunID(gomock.Any(), deadID).Return(events, nil).Times(1)

			var heirID string
			var heirRun sqlc.InsertAgentRunParams
			runRepo.EXPECT().Insert(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, p sqlc.InsertAgentRunParams) *apierror.APIError {
					heirID, heirRun = p.ID, p
					return nil
				}).Times(1)

			var copied []sqlc.InsertAgentRunEventParams
			eventRepo.EXPECT().Insert(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, p sqlc.InsertAgentRunEventParams) *apierror.APIError {
					copied = append(copied, p)
					return nil
				}).Times(wantCopied)

			runRepo.EXPECT().UpdateStarted(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, gotID string) (int64, *apierror.APIError) {
					if gotID != heirID {
						t.Errorf("UpdateStarted id = %q, want heir %q", gotID, heirID)
					}
					return 1, nil
				}).Times(1)

			s := &agentDefSvcImpl{repos: factory, txManager: fakeTxManager{f: factory}}
			continued, apiErr := s.continueChatRun(context.Background(), domain.ChatRunInput{
				AccountID:        acct,
				ContinueRunID:    deadID,
				Message:          reply,
				TriggerMessageID: triggerID,
			})
			if apiErr != nil {
				t.Fatalf("continueChatRun error: %v", apiErr)
			}
			if !continued {
				t.Fatalf("continued = false, want true (should fork an heir run)")
			}

			// Heir inherits the dead run's identity and is conversation-linked to the reply.
			if heirID == deadID || heirID == "" {
				t.Errorf("heir id = %q, want a fresh id distinct from the dead run", heirID)
			}
			if heirRun.StatusCode != domain.RunStatusPending {
				t.Errorf("heir status = %q, want pending (UpdateStarted promotes it to running)", heirRun.StatusCode)
			}
			if heirRun.AgentDefinitionID != defID || heirRun.AgentConfigID.String != configID {
				t.Errorf("heir agent identity = (%q,%q), want (%q,%q)", heirRun.AgentDefinitionID, heirRun.AgentConfigID.String, defID, configID)
			}
			if heirRun.ConversationID.String != convID || heirRun.TriggerMessageID.String != triggerID {
				t.Errorf("heir conversation link = (%q,%q), want (%q,%q)", heirRun.ConversationID.String, heirRun.TriggerMessageID.String, convID, triggerID)
			}

			// Transcript copied minus failure markers, re-sequenced from 0, action links cleared.
			if len(copied) != wantCopied {
				t.Fatalf("copied %d events, want %d", len(copied), wantCopied)
			}
			for i, e := range copied {
				if e.StepType == stepTypeError || e.StepType == stepTypeCancelled {
					t.Errorf("event %d step %q should have been dropped", i, e.StepType)
				}
				if e.AgentRunID != heirID || e.AccountID != acct {
					t.Errorf("event %d run/account = (%q,%q), want (%q,%q)", i, e.AgentRunID, e.AccountID, heirID, acct)
				}
				if int(e.Sequence) != i {
					t.Errorf("event %d sequence = %d, want %d (re-sequenced from 0)", i, e.Sequence, i)
				}
				if e.AgentActionID.Valid {
					t.Errorf("event %d kept agent_action_id %q; it belongs to the dead run", i, e.AgentActionID.String)
				}
			}

			// The reply is enqueued as the heir's next (continue) turn, threaded back to the conversation.
			if len(outbox.inputs) != 1 {
				t.Fatalf("enqueued %d outbox messages, want 1", len(outbox.inputs))
			}
			if got := outbox.inputs[0].MessageType; got != string(contracts.AgentCmdContinueRun) {
				t.Errorf("outbox message type = %q, want %q", got, contracts.AgentCmdContinueRun)
			}
			var cont messaging.AgentContinueRunData
			if err := json.Unmarshal(outbox.inputs[0].Payload.Data, &cont); err != nil {
				t.Fatalf("unmarshal continue data: %v", err)
			}
			if cont.AgentRunID != heirID || cont.Message != reply || cont.ReplyToMessageID != triggerID {
				t.Errorf("continue data = %+v, want heir=%q message=%q replyTo=%q", cont, heirID, reply, triggerID)
			}
		})
	}
}
