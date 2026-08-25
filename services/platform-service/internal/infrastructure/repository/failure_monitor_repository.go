package repository

import (
	"context"

	"github.com/open-mrp/api/services/platform-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/db"
	"github.com/open-mrp/api/shared/messaging"
	"github.com/open-mrp/api/shared/tracing"
)

var failureMonitorRepoTracer = tracing.GetTracer("platform-service.failure_monitor_repository")

type failureMonitorRepoImpl struct {
	queries *sqlc.Queries
}

// NewFailureMonitorRepo creates the repository the message failure monitor uses to scan the shared message_inbox and message_outbox tables for un-alerted failures and mark them alerted.
func NewFailureMonitorRepo(queries *sqlc.Queries) messaging.FailureMonitorRepo {
	return &failureMonitorRepoImpl{queries: queries}
}

func (r *failureMonitorRepoImpl) ListUnalertedInboxFailures(ctx context.Context, crashStuckMinutes int, limit int32) ([]messaging.InboxFailure, error) {
	ctx, span := failureMonitorRepoTracer.Start(ctx, "repository.failure_monitor.list_inbox_failures")
	defer span.End()

	rows, err := r.queries.ListUnalertedInboxFailures(ctx, sqlc.ListUnalertedInboxFailuresParams{
		CrashStuckMinutes: crashStuckMinutes,
		Limit:             limit,
	})
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	failures := make([]messaging.InboxFailure, 0, len(rows))
	for _, row := range rows {
		failures = append(failures, messaging.InboxFailure{
			ID:          row.ID,
			MessageID:   row.MessageID,
			ServiceName: row.ServiceName,
			Handler:     row.Handler,
			MessageType: row.MessageType,
			Attempts:    int(row.Attempts),
			LastError:   db.StringFromNullString(row.LastError),
			ReceivedAt:  row.ReceivedAt,
		})
	}

	return failures, nil
}

func (r *failureMonitorRepoImpl) ListUnalertedOutboxFailures(ctx context.Context, limit int32) ([]messaging.OutboxFailure, error) {
	ctx, span := failureMonitorRepoTracer.Start(ctx, "repository.failure_monitor.list_outbox_failures")
	defer span.End()

	rows, err := r.queries.ListUnalertedOutboxFailures(ctx, limit)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	failures := make([]messaging.OutboxFailure, 0, len(rows))
	for _, row := range rows {
		failures = append(failures, messaging.OutboxFailure{
			ID:          row.ID,
			MessageID:   row.MessageID,
			ServiceName: row.ServiceName,
			MessageType: row.MessageType,
			Destination: row.Destination,
			RoutingKey:  row.RoutingKey.String,
			Attempts:    int(row.Attempts),
			MaxAttempts: int(row.MaxAttempts),
			LastError:   db.StringFromNullString(row.LastError),
			CreatedAt:   row.CreatedAt,
		})
	}

	return failures, nil
}

func (r *failureMonitorRepoImpl) MarkInboxAlerted(ctx context.Context, ids []int64) error {
	ctx, span := failureMonitorRepoTracer.Start(ctx, "repository.failure_monitor.mark_inbox_alerted")
	defer span.End()

	if len(ids) == 0 {
		return nil
	}

	if err := r.queries.MarkInboxRecordsAlerted(ctx, ids); err != nil {
		span.RecordError(err)
		return err
	}

	return nil
}

func (r *failureMonitorRepoImpl) MarkOutboxAlerted(ctx context.Context, ids []int64) error {
	ctx, span := failureMonitorRepoTracer.Start(ctx, "repository.failure_monitor.mark_outbox_alerted")
	defer span.End()

	if len(ids) == 0 {
		return nil
	}

	if err := r.queries.MarkOutboxRecordsAlerted(ctx, ids); err != nil {
		span.RecordError(err)
		return err
	}

	return nil
}
