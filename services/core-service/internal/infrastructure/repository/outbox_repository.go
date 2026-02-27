package repository

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/augno/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/contracts"
	"github.com/augno/api/shared/db"
	"github.com/augno/api/shared/id"
	"github.com/augno/api/shared/messaging"
	"github.com/augno/api/shared/tracing"
)

var outboxRepoTracer = tracing.GetTracer("core-service.outbox_repository")

type outboxRepoImpl struct {
	queries *sqlc.Queries
}

func NewOutboxRepo(queries *sqlc.Queries) messaging.OutboxRepo {
	return &outboxRepoImpl{queries: queries}
}

func (r *outboxRepoImpl) Create(ctx context.Context, input messaging.OutboxMessageInput) (int64, error) {
	ctx, span := outboxRepoTracer.Start(ctx, "repository.outbox.create")
	defer span.End()

	length := id.IDLength22

	messageID := input.MessageID
	if messageID == "" {
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

	result, err := r.queries.CreateOutboxMessage(ctx, sqlc.CreateOutboxMessageParams{
		MessageID:       messageID,
		ServiceName:     input.ServiceName,
		MessageType:     input.MessageType,
		Destination:     input.Destination,
		RoutingKey:      db.NullString(input.RoutingKey),
		Headers:         nil,
		Payload:         payloadJSON,
		MaxAttempts:     int32(maxAttempts), // #nosec G115 - small config value
		RequestID:       db.NullString(input.Payload.RequestID),
		ParentMessageID: db.NullString(input.Payload.ParentMessageID),
	})

	if err != nil {
		span.RecordError(err)
		return 0, err
	}

	return result, nil
}

type outboxEnqueuerRepoImpl struct {
	dbPool  *sql.DB
	queries *sqlc.Queries
}

func NewOutboxEnqueuerRepo(dbPool *sql.DB, queries *sqlc.Queries) messaging.OutboxEnqueuerRepo {
	return &outboxEnqueuerRepoImpl{dbPool: dbPool, queries: queries}
}

func (r *outboxEnqueuerRepoImpl) AcquireAndLock(ctx context.Context, lockOwner string, limit int, lockDurationSeconds int) ([]*messaging.OutboxMessage, error) {
	ctx, span := tracing.StartSpan(ctx, outboxRepoTracer, "repository.outbox.acquire_and_lock")
	defer span.End()

	tx, err := r.dbPool.BeginTx(ctx, nil)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}
	defer tx.Rollback()

	txQueries := r.queries.WithTx(tx)

	err = txQueries.AcquireOutboxMessages(ctx, sqlc.AcquireOutboxMessagesParams{
		LockOwner: db.NullString(lockOwner),
		DATEADD:   lockDurationSeconds,
		Limit:     int32(limit), // #nosec G115 - small config value
	})
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	rows, err := txQueries.GetLockedOutboxMessages(ctx, db.NullString(lockOwner))
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	if err := tx.Commit(); err != nil {
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
			NextRunAt:       row.NextRunAt,
			LockedAt:        db.TimeFromNullTime(row.LockedAt),
			LockOwner:       db.StringFromNullString(row.LockOwner),
			LockExpiresAt:   db.TimeFromNullTime(row.LockExpiresAt),
			LastError:       db.StringFromNullString(row.LastError),
			PublishedAt:     db.TimeFromNullTime(row.PublishedAt),
			RequestID:       db.StringFromNullString(row.RequestID),
			ParentMessageID: db.StringFromNullString(row.ParentMessageID),
			CreatedAt:       row.CreatedAt,
			UpdatedAt:       row.UpdatedAt,
		}
		messages = append(messages, msg)
	}

	return messages, nil
}

func (r *outboxEnqueuerRepoImpl) MarkPublished(ctx context.Context, id int64) error {
	ctx, span := tracing.StartSpan(ctx, outboxRepoTracer, "repository.outbox.mark_published")
	defer span.End()

	err := r.queries.MarkOutboxMessagePublished(ctx, id)
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
		LastError: db.NullString(errorMsg),
		DATEADD:   retryDelaySecs,
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

	result, err := r.queries.CleanupExpiredOutboxLocks(ctx, limit)
	if err != nil {
		span.RecordError(err)
		return 0, err
	}

	count, err := result.RowsAffected()
	if err != nil {
		span.RecordError(err)
		return 0, err
	}

	return count, nil
}

func (r *outboxEnqueuerRepoImpl) PurgePublished(ctx context.Context, retentionHours int, limit int32) (int64, error) {
	ctx, span := tracing.StartSpan(ctx, outboxRepoTracer, "repository.outbox.purge_published")
	defer span.End()

	result, err := r.queries.PurgePublishedOutboxMessages(ctx, sqlc.PurgePublishedOutboxMessagesParams{
		DATESUB: retentionHours,
		Limit:   limit,
	})
	if err != nil {
		span.RecordError(err)
		return 0, err
	}

	count, err := result.RowsAffected()
	if err != nil {
		span.RecordError(err)
		return 0, err
	}

	return count, nil
}
