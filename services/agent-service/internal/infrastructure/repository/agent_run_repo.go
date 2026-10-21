package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/open-mrp/api/services/agent-service/internal/domain"
	"github.com/open-mrp/api/services/agent-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/tracing"
	"go.opentelemetry.io/otel/trace"
)

var runRepoTracer = tracing.GetTracer("agent-service.agent_run_repository")

type agentRunRepoImpl struct {
	queries *sqlc.Queries
}

func NewAgentRunRepo(queries *sqlc.Queries) *agentRunRepoImpl {
	return &agentRunRepoImpl{queries: queries}
}

func (r *agentRunRepoImpl) Insert(ctx context.Context, params sqlc.InsertAgentRunParams) *apierror.APIError {
	ctx, span := tracing.StartSpan(ctx, runRepoTracer, "repository.agent_run.insert")
	defer span.End()
	if apiErr := db.MapSQLError(r.queries.InsertAgentRun(ctx, params)); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *agentRunRepoImpl) GetByID(ctx context.Context, id string) (*sqlc.AgentRun, *apierror.APIError) {
	ctx, span := tracing.StartSpan(ctx, runRepoTracer, "repository.agent_run.get_by_id")
	defer span.End()
	row, err := r.queries.GetAgentRunByID(ctx, id)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return &row, nil
}

func (r *agentRunRepoImpl) ListByAccountFiltered(ctx context.Context, params domain.AgentRunPageQuery) ([]sqlc.AgentRun, *apierror.APIError) {
	ctx, span := tracing.StartSpan(ctx, runRepoTracer, "repository.agent_run.list_by_account_filtered")
	defer span.End()
	query, args := buildAgentRunListQuery(params)
	rows, err := queryList(ctx, r.queries.DB(), query, args, scanAgentRun)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return rows, nil
}

func (r *agentRunRepoImpl) UpdateStatus(ctx context.Context, id, status string) *apierror.APIError {
	ctx, span := tracing.StartSpan(ctx, runRepoTracer, "repository.agent_run.update_status")
	defer span.End()
	if apiErr := db.MapSQLError(r.queries.UpdateAgentRunStatus(ctx, sqlc.UpdateAgentRunStatusParams{
		StatusCode: status,
		ID:         id,
	})); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *agentRunRepoImpl) MarkCancelledByUser(ctx context.Context, id string) *apierror.APIError {
	ctx, span := tracing.StartSpan(ctx, runRepoTracer, "repository.agent_run.mark_cancelled_by_user")
	defer span.End()
	if apiErr := db.MapSQLError(r.queries.MarkAgentRunCancelledByUser(ctx, id)); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

// MarkRetrying atomically transitions a failed run back to running and increments its retry counter, returning the new count. The underlying query guards on status='failed', so a run that isn't failed
// (already retried, or never failed) yields no rows — surfaced here as a not-found error for the caller to treat as "not retryable".
func (r *agentRunRepoImpl) MarkRetrying(ctx context.Context, id string) (int32, *apierror.APIError) {
	ctx, span := tracing.StartSpan(ctx, runRepoTracer, "repository.agent_run.mark_retrying")
	defer span.End()
	affected, err := r.queries.MarkAgentRunRetrying(ctx, id)
	if err != nil {
		return 0, tracing.Trace(span, db.MapSQLError(err))
	}
	return r.retryCountAfterMark(ctx, span, id, affected)
}

// MarkAutoRetrying increments a still-running run's retry counter (clearing any stale error) so the runner can transparently re-enqueue it after a transient, whole-chain-unavailable failure. The underlying query guards on status='running' and leaves the status running, so a run that already left the running state yields no rows — surfaced here as a not-found error for the caller to treat as "not retryable".
func (r *agentRunRepoImpl) MarkAutoRetrying(ctx context.Context, id string) (int32, *apierror.APIError) {
	ctx, span := tracing.StartSpan(ctx, runRepoTracer, "repository.agent_run.mark_auto_retrying")
	defer span.End()
	affected, err := r.queries.MarkAgentRunAutoRetrying(ctx, id)
	if err != nil {
		return 0, tracing.Trace(span, db.MapSQLError(err))
	}
	return r.retryCountAfterMark(ctx, span, id, affected)
}

func (r *agentRunRepoImpl) retryCountAfterMark(ctx context.Context, span trace.Span, id string, affected int64) (int32, *apierror.APIError) {
	if affected == 0 {
		return 0, tracing.Trace(span, db.MapSQLError(sql.ErrNoRows))
	}
	count, err := r.queries.GetAgentRunRetryCount(ctx, id)
	if err != nil {
		return 0, tracing.Trace(span, db.MapSQLError(err))
	}
	return count, nil
}

// UpdateStarted atomically claims a pending run for execution, flipping it to 'running' and stamping started_at. The query guards on status='pending', so the returned row count is 1 for the delivery that wins the claim and 0 for a duplicate (already-started, completed, or cancelled) run — letting the caller drop a redelivered execute_run command instead of running it a second time.
func (r *agentRunRepoImpl) UpdateStarted(ctx context.Context, id string) (int64, *apierror.APIError) {
	ctx, span := tracing.StartSpan(ctx, runRepoTracer, "repository.agent_run.update_started")
	defer span.End()
	rows, err := r.queries.UpdateAgentRunStarted(ctx, id)
	if err != nil {
		return 0, tracing.Trace(span, db.MapSQLError(err))
	}
	return rows, nil
}

func (r *agentRunRepoImpl) UpdateCompleted(ctx context.Context, params sqlc.UpdateAgentRunCompletedParams) *apierror.APIError {
	ctx, span := tracing.StartSpan(ctx, runRepoTracer, "repository.agent_run.update_completed")
	defer span.End()
	params.IsCompleted = params.StatusCode == domain.RunStatusCompleted
	if apiErr := db.MapSQLError(r.queries.UpdateAgentRunCompleted(ctx, params)); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *agentRunRepoImpl) UpdateCancelled(ctx context.Context, params sqlc.UpdateAgentRunCancelledParams) *apierror.APIError {
	ctx, span := tracing.StartSpan(ctx, runRepoTracer, "repository.agent_run.update_cancelled")
	defer span.End()
	if apiErr := db.MapSQLError(r.queries.UpdateAgentRunCancelled(ctx, params)); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *agentRunRepoImpl) UpdateFailed(ctx context.Context, params sqlc.UpdateAgentRunFailedParams) *apierror.APIError {
	ctx, span := tracing.StartSpan(ctx, runRepoTracer, "repository.agent_run.update_failed")
	defer span.End()
	if apiErr := db.MapSQLError(r.queries.UpdateAgentRunFailed(ctx, params)); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

// ReapStalledRuns fails every run that has been 'running' since before cutoff, returning the reaped run ids. It is the safety net for runs orphaned by a process kill mid-flight (the in-process goroutine that would have finalized them is gone), which would otherwise sit in 'running' forever.
func (r *agentRunRepoImpl) ReapStalledRuns(ctx context.Context, cutoff time.Time, errorMessage string) ([]string, *apierror.APIError) {
	ctx, span := tracing.StartSpan(ctx, runRepoTracer, "repository.agent_run.reap_stalled_runs")
	defer span.End()
	stalled, err := r.queries.ListStalledRuns(ctx, db.NullTime(cutoff))
	if err != nil {
		return nil, tracing.Trace(span, db.MapSQLError(err))
	}
	ids := make([]string, 0, len(stalled))
	for _, run := range stalled {
		affected, err := r.queries.ReapStalledRun(ctx, sqlc.ReapStalledRunParams{
			ErrorMessage: db.NullString(errorMessage),
			ID:           run.ID,
			StartedAt:    db.NullTime(cutoff),
		})
		if err != nil {
			return ids, tracing.Trace(span, db.MapSQLError(err))
		}
		if affected > 0 {
			ids = append(ids, run.ID)
		}
	}
	return ids, nil
}

func (r *agentRunRepoImpl) MarkDivergedFromConversation(ctx context.Context, id string) *apierror.APIError {
	ctx, span := tracing.StartSpan(ctx, runRepoTracer, "repository.agent_run.mark_diverged_from_conversation")
	defer span.End()
	if apiErr := db.MapSQLError(r.queries.MarkAgentRunDivergedFromConversation(ctx, id)); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *agentRunRepoImpl) GetLastByConfigID(ctx context.Context, configID string) (*sqlc.AgentRun, *apierror.APIError) {
	ctx, span := tracing.StartSpan(ctx, runRepoTracer, "repository.agent_run.get_last_by_config_id")
	defer span.End()
	row, err := r.queries.GetLastRunByConfigID(ctx, db.NullString(configID))
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return &row, nil
}
