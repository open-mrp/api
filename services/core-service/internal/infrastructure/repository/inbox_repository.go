package repository

import (
	"context"

	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/db"
	"github.com/open-mrp/api/shared/messaging"
	"github.com/open-mrp/api/shared/tracing"
)

var inboxRepoTracer = tracing.GetTracer("core-service.inbox_repository")

type inboxRepoImpl struct {
	queries *sqlc.Queries
}

func NewInboxRepo(queries *sqlc.Queries) messaging.InboxRepo {
	return &inboxRepoImpl{queries: queries}
}

func NewInboxPurgerRepo(queries *sqlc.Queries) messaging.InboxPurgerRepo {
	return &inboxRepoImpl{queries: queries}
}

func (r *inboxRepoImpl) PurgeProcessed(ctx context.Context, retentionHours int, limit int32) (int64, error) {
	ctx, span := inboxRepoTracer.Start(ctx, "repository.inbox.purge_processed")
	defer span.End()

	result, err := r.queries.PurgeProcessedInboxMessages(ctx, sqlc.PurgeProcessedInboxMessagesParams{
		Column1: retentionHours,
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

func (r *inboxRepoImpl) TryInsert(ctx context.Context, input messaging.InboxRecordInput) (int64, error) {
	ctx, span := inboxRepoTracer.Start(ctx, "repository.inbox.try_insert")
	defer span.End()

	result, err := r.queries.TryInsertInboxRecord(ctx, sqlc.TryInsertInboxRecordParams{
		MessageID:           input.MessageID,
		ServiceName:         input.ServiceName,
		Handler:             input.Handler,
		MessageType:         input.MessageType,
		RequestID:           db.NullString(input.RequestID),
		ParentMessageID:     db.NullString(input.ParentMessageID),
		LockOwner:           db.NullString(input.LockOwner),
		LockDurationSeconds: input.LockTTLSeconds,
	})
	if err != nil {
		span.RecordError(err)
		return 0, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		span.RecordError(err)
		return 0, err
	}

	return id, nil
}

func (r *inboxRepoImpl) GetByMessageAndHandler(ctx context.Context, messageID, handler string) (*messaging.InboxRecord, error) {
	ctx, span := inboxRepoTracer.Start(ctx, "repository.inbox.get_by_message_and_handler")
	defer span.End()

	row, err := r.queries.GetInboxRecordByMessageAndHandler(ctx, sqlc.GetInboxRecordByMessageAndHandlerParams{
		MessageID: messageID,
		Handler:   handler,
	})
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	record := &messaging.InboxRecord{
		ID:              row.ID,
		MessageID:       row.MessageID,
		ServiceName:     row.ServiceName,
		Handler:         row.Handler,
		MessageType:     row.MessageType,
		RequestID:       db.StringFromNullString(row.RequestID),
		ParentMessageID: db.StringFromNullString(row.ParentMessageID),
		Status:          messaging.InboxStatus(row.Status),
		Attempts:        int(row.Attempts),
		LastError:       db.StringFromNullString(row.LastError),
		ReceivedAt:      row.ReceivedAt,
		ProcessedAt:     db.TimeFromNullTime(row.ProcessedAt),
		FailedAt:        db.TimeFromNullTime(row.FailedAt),
		LockOwner:       db.StringFromNullString(row.LockOwner),
		LockExpiresAt:   db.TimeFromNullTime(row.LockExpiresAt),
	}

	return record, nil
}

func (r *inboxRepoImpl) Claim(ctx context.Context, id int64, owner string, ttlSeconds int) (bool, error) {
	ctx, span := inboxRepoTracer.Start(ctx, "repository.inbox.claim")
	defer span.End()

	rows, err := r.queries.ClaimInboxRecord(ctx, sqlc.ClaimInboxRecordParams{
		ID:                  id,
		LockOwner:           db.NullString(owner),
		LockDurationSeconds: ttlSeconds,
	})
	if err != nil {
		span.RecordError(err)
		return false, err
	}

	return rows > 0, nil
}

func (r *inboxRepoImpl) Complete(ctx context.Context, id int64, owner string) (bool, error) {
	ctx, span := inboxRepoTracer.Start(ctx, "repository.inbox.complete")
	defer span.End()

	rows, err := r.queries.CompleteInboxRecord(ctx, sqlc.CompleteInboxRecordParams{
		ID:        id,
		LockOwner: db.NullString(owner),
	})
	if err != nil {
		span.RecordError(err)
		return false, err
	}

	return rows > 0, nil
}

func (r *inboxRepoImpl) MarkFailed(ctx context.Context, id int64, owner, errMsg string) error {
	ctx, span := inboxRepoTracer.Start(ctx, "repository.inbox.mark_failed")
	defer span.End()

	_, err := r.queries.MarkInboxRecordFailed(ctx, sqlc.MarkInboxRecordFailedParams{
		ID:        id,
		LastError: db.NullString(errMsg),
		LockOwner: db.NullString(owner),
	})
	if err != nil {
		span.RecordError(err)
		return err
	}

	return nil
}

func (r *inboxRepoImpl) MarkDiscarded(ctx context.Context, id int64, owner, reason string) error {
	ctx, span := inboxRepoTracer.Start(ctx, "repository.inbox.mark_discarded")
	defer span.End()

	_, err := r.queries.MarkInboxRecordDiscarded(ctx, sqlc.MarkInboxRecordDiscardedParams{
		ID:        id,
		LastError: db.NullString(reason),
		LockOwner: db.NullString(owner),
	})
	if err != nil {
		span.RecordError(err)
		return err
	}

	return nil
}

func (r *inboxRepoImpl) MarkIgnored(ctx context.Context, id int64, owner, reason string) error {
	ctx, span := inboxRepoTracer.Start(ctx, "repository.inbox.mark_ignored")
	defer span.End()

	_, err := r.queries.MarkInboxRecordIgnored(ctx, sqlc.MarkInboxRecordIgnoredParams{
		ID:        id,
		LastError: db.NullString(reason),
		LockOwner: db.NullString(owner),
	})
	if err != nil {
		span.RecordError(err)
		return err
	}

	return nil
}
