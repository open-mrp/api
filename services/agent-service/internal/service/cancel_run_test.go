package service

import (
	"context"
	"testing"

	"github.com/open-mrp/api/services/agent-service/internal/domain"
	factorymock "github.com/open-mrp/api/services/agent-service/internal/domain/mock/factory"
	repositorymock "github.com/open-mrp/api/services/agent-service/internal/domain/mock/repository"
	"github.com/open-mrp/api/services/agent-service/internal/infrastructure/sqlc"
	apierror "github.com/open-mrp/api/shared/errors"
	"go.uber.org/mock/gomock"
)

// runCancelled is the cooperative-cancellation gate the agent loop polls between model calls and tool
// calls: it reports true only when the run's DB status has been flipped to cancelled out-of-band.
func TestRunCancelled(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		run    *sqlc.AgentRun
		runErr *apierror.APIError
		want   bool
	}{
		{name: "running is not cancelled", run: &sqlc.AgentRun{ID: "agr_1", StatusCode: domain.RunStatusRunning}, want: false},
		{name: "cancelled stops the loop", run: &sqlc.AgentRun{ID: "agr_1", StatusCode: domain.RunStatusCancelled}, want: true},
		// A transient read failure must not abort a healthy run.
		{name: "read error is treated as not cancelled", run: nil, runErr: apierror.NewInternalError(nil, "boom"), want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			runRepo := repositorymock.NewMockAgentRunRepo(ctrl)
			runRepo.EXPECT().GetByID(gomock.Any(), "agr_1").Return(tc.run, tc.runErr).Times(1)
			factory := factorymock.NewMockRepoFactory(ctrl)
			factory.EXPECT().NewAgentRunRepo().Return(runRepo).Times(1)

			s := &runnerSvc{repos: factory}
			if got := s.runCancelled(context.Background(), "agr_1"); got != tc.want {
				t.Errorf("runCancelled = %v, want %v", got, tc.want)
			}
		})
	}
}

// cancelledResult returns a non-error, terminal result flagged Cancelled so the caller finalizes the
// run as cancelled (rather than completed/awaiting_input) while still carrying any partial work.
func TestCancelledResult(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	eventRepo := repositorymock.NewMockAgentRunEventRepo(ctrl)
	eventRepo.EXPECT().Insert(gomock.Any(), gomock.Any()).Return(nil).Times(1)
	factory := factorymock.NewMockRepoFactory(ctrl)
	factory.EXPECT().NewAgentRunEventRepo().Return(eventRepo).Times(1)

	s := &runnerSvc{repos: factory} // no broker → emitEvent's publish is skipped, only the DB insert runs
	runCtx := &domain.HandlerRunContext{
		Actions: []domain.PendingAction{{ToolSlug: "search"}},
	}
	seq := 0
	res := s.cancelledResult(context.Background(), &sqlc.AgentRun{ID: "agr_1"}, "acct_1", &seq, runCtx, 12, 3, "anthropic", "claude-x")
	if !res.Cancelled {
		t.Error("expected Cancelled to be true")
	}
	if res.InputTokens != 12 || res.OutputTokens != 3 {
		t.Errorf("expected partial token usage preserved, got in=%d out=%d", res.InputTokens, res.OutputTokens)
	}
	if len(res.Actions) != 1 {
		t.Errorf("expected partial actions preserved, got %d", len(res.Actions))
	}
}
