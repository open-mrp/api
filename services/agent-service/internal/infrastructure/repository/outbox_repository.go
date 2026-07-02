package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	agentdb "github.com/augno/api/services/agent-service/internal/infrastructure/db"
	"github.com/augno/api/services/agent-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/contracts"
	"github.com/augno/api/shared/id"
	"github.com/augno/api/shared/messaging"
	"github.com/augno/api/shared/tracing"
	"github.com/jackc/pgx/v5/pgxpool"
)

var outboxRepoTracer = tracing.GetTracer("agent-service.outbox_repository")

type outboxRepoImpl struct {
	queries *sqlc.Queries
}

func NewOutboxRepo(queries *sqlc.Queries) messaging.OutboxRepo {
	return &outboxRepoImpl{queries: queries}
}

func (r *outboxRepoImpl) Create(ctx context.Context, input messaging.OutboxMessageInput) (int64, error) {
	ctx, span := tracing.StartSpan(ctx, outboxRepoTracer, "repository.outbox.create")
	defer span.End()

	messageID := input.MessageID
	if messageID == "" {
		length := id.IDLength22
		genID, err := id.GenID(id.MessageIDPrefix, &length)
		if err != nil {
			span.RecordError(err)
			return 0, err
		}
		messageID = genID
	}

	payloadJSON, err := json.Marshal(input.Payload)
	if err != nil {
		span.RecordError(err)
		return 0, err
	}

	maxAttempts := input.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 25
	}

	// Defer first delivery by DelaySeconds (0 = available now). Column9 is the next_run_at interval offset; it is rendered server-side as now() + ('<n>' || ' seconds')::interval, mirroring the enqueuer's retry-backoff scheduling so a re-enqueued message is not republished into a still-failing dependency.
	delaySecs := input.DelaySeconds
	if delaySecs < 0 {
		delaySecs = 0
	}

	result, err := r.queries.CreateOutboxMessage(ctx, sqlc.CreateOutboxMessageParams{
		MessageID:       messageID,
		ServiceName:     input.ServiceName,
		MessageType:     input.MessageType,
		Destination:     input.Destination,
		RoutingKey:      agentdb.PgText(input.RoutingKey),
		Headers:         nil,
		Payload:         payloadJSON,
		MaxAttempts:     int32(maxAttempts), // #nosec G115 - small config value
		Column9:         agentdb.PgText(strconv.Itoa(delaySecs)),
		RequestID:       agentdb.PgText(input.Payload.RequestID),
		ParentMessageID: agentdb.PgText(input.Payload.ParentMessageID),
	})
	if err != nil {
		span.RecordError(err)
		return 0, err
	}

	return result, nil
}

type outboxEnqueuerRepoImpl struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

func NewOutboxEnqueuerRepo(pool *pgxpool.Pool, queries *sqlc.Queries) messaging.OutboxEnqueuerRepo {
	return &outboxEnqueuerRepoImpl{pool: pool, queries: queries}
}

func (r *outboxEnqueuerRepoImpl) AcquireAndLock(ctx context.Context, lockOwner string, limit int, lockDurationSeconds int) ([]*messaging.OutboxMessage, error) {
	ctx, span := tracing.StartSpan(ctx, outboxRepoTracer, "repository.outbox.acquire_and_lock")
	defer span.End()

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}
	defer tx.Rollback(ctx)

	txQueries := r.queries.WithTx(tx)

	err = txQueries.AcquireOutboxMessages(ctx, sqlc.AcquireOutboxMessagesParams{
		LockOwner: agentdb.PgText(lockOwner),
		Column2:   agentdb.PgText(fmt.Sprintf("%d", lockDurationSeconds)),
		Limit:     int32(limit), // #nosec G115 - small config value
	})
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	rows, err := txQueries.GetLockedOutboxMessages(ctx, agentdb.PgText(lockOwner))
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		span.RecordError(err)
		return nil, err
	}

	messages := make([]*messaging.OutboxMessage, 0, len(rows))
	for _, row := range rows {
		var payload contracts.AmqpMessage
		if err := json.Unmarshal(row.Payload, &payload); err != nil {
			span.RecordError(err)
			return nil, err
		}

		msg := &messaging.OutboxMessage{
			ID:              row.ID,
			MessageID:       row.MessageID,
			ServiceName:     row.ServiceName,
			MessageType:     row.MessageType,
			Destination:     row.Destination,
			RoutingKey:      row.RoutingKey.String,
			Payload:         payload,
			Status:          messaging.OutboxStatus(row.Status),
			Attempts:        int(row.Attempts),
			MaxAttempts:     int(row.MaxAttempts),
			NextRunAt:       row.NextRunAt.Time,
			LockedAt:        agentdb.TimeFromPgTimestamptz(row.LockedAt),
			LockOwner:       agentdb.StringFromPgText(row.LockOwner),
			LockExpiresAt:   agentdb.TimeFromPgTimestamptz(row.LockExpiresAt),
			LastError:       agentdb.StringFromPgText(row.LastError),
			PublishedAt:     agentdb.TimeFromPgTimestamptz(row.PublishedAt),
			RequestID:       agentdb.StringFromPgText(row.RequestID),
			ParentMessageID: agentdb.StringFromPgText(row.ParentMessageID),
			CreatedAt:       row.CreatedAt.Time,
			UpdatedAt:       row.UpdatedAt.Time,
		}
		messages = append(messages, msg)
	}

	return messages, nil
}

func (r *outboxEnqueuerRepoImpl) MarkPublished(ctx context.Context, ids []int64) error {
	ctx, span := tracing.StartSpan(ctx, outboxRepoTracer, "repository.outbox.mark_published")
	defer span.End()

	if len(ids) == 0 {
		return nil
	}

	err := r.queries.MarkOutboxMessagesPublished(ctx, ids)
	if err != nil {
		span.RecordError(err)
		return err
	}

	return nil
}

func (r *outboxEnqueuerRepoImpl) MarkFailed(ctx context.Context, id int64, errorMsg string, retryDelaySecs int) error {
	ctx, span := tracing.StartSpan(ctx, outboxRepoTracer, "repository.outbox.mark_failed")
	defer span.End()

	err := r.queries.MarkOutboxMessageFailed(ctx, sqlc.MarkOutboxMessageFailedParams{
		ID:        id,
		LastError: agentdb.PgText(errorMsg),
		Column2:   agentdb.PgText(fmt.Sprintf("%d", retryDelaySecs)),
	})
	if err != nil {
		span.RecordError(err)
		return err
	}

	return nil
}

func (r *outboxEnqueuerRepoImpl) CleanupExpiredLocks(ctx context.Context, limit int32) (int64, error) {
	ctx, span := tracing.StartSpan(ctx, outboxRepoTracer, "repository.outbox.cleanup_expired_locks")
	defer span.End()

	count, err := r.queries.CleanupExpiredOutboxLocks(ctx, limit)
	if err != nil {
		span.RecordError(err)
		return 0, err
	}

	return count, nil
}

func (r *outboxEnqueuerRepoImpl) PurgePublished(ctx context.Context, retentionHours int, limit int32) (int64, error) {
	ctx, span := tracing.StartSpan(ctx, outboxRepoTracer, "repository.outbox.purge_published")
	defer span.End()

	count, err := r.queries.PurgePublishedOutboxMessages(ctx, sqlc.PurgePublishedOutboxMessagesParams{
		Column1: agentdb.PgText(fmt.Sprintf("%d", retentionHours)),
		Limit:   limit,
	})
	if err != nil {
		span.RecordError(err)
		return 0, err
	}

	return count, nil
}
