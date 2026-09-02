package repository

import (
	"context"
	"fmt"

	agentdb "github.com/open-mrp/api/services/agent-service/internal/infrastructure/db"
	"github.com/open-mrp/api/services/agent-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/messaging"
	"github.com/open-mrp/api/shared/tracing"
)

var inboxRepoTracer = tracing.GetTracer("agent-service.inbox_repository")

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

	count, err := r.queries.PurgeProcessedInboxMessages(ctx, sqlc.PurgeProcessedInboxMessagesParams{
		Column1: agentdb.PgText(fmt.Sprintf("%d", retentionHours)),
		Limit:   limit,
	})
	if err != nil {
		span.RecordError(err)
		return 0, err
	}

	return count, nil
}

func (r *inboxRepoImpl) TryInsert(ctx context.Context, input messaging.InboxRecordInput) (int64, error) {
	ctx, span := inboxRepoTracer.Start(ctx, "repository.inbox.try_insert")
	defer span.End()

	id, err := r.queries.TryInsertInboxRecord(ctx, sqlc.TryInsertInboxRecordParams{
		MessageID:       input.MessageID,
		ServiceName:     input.ServiceName,
		Handler:         input.Handler,
		MessageType:     input.MessageType,
		RequestID:       agentdb.PgText(input.RequestID),
		ParentMessageID: agentdb.PgText(input.ParentMessageID),
		LockOwner:       agentdb.PgText(input.LockOwner),
		Column8:         agentdb.PgText(fmt.Sprintf("%d", input.LockTTLSeconds)),
	})
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
		RequestID:       agentdb.StringFromPgText(row.RequestID),
		ParentMessageID: agentdb.StringFromPgText(row.ParentMessageID),
		Status:          messaging.InboxStatus(row.Status),
		Attempts:        int(row.Attempts),
		LastError:       agentdb.StringFromPgText(row.LastError),
		ReceivedAt:      row.ReceivedAt.Time,
		ProcessedAt:     agentdb.TimeFromPgTimestamptz(row.ProcessedAt),
		FailedAt:        agentdb.TimeFromPgTimestamptz(row.FailedAt),
		LockOwner:       agentdb.StringFromPgText(row.LockOwner),
		LockExpiresAt:   agentdb.TimeFromPgTimestamptz(row.LockExpiresAt),
	}

	return record, nil
}

func (r *inboxRepoImpl) Claim(ctx context.Context, id int64, owner string, ttlSeconds int) (bool, error) {
	ctx, span := inboxRepoTracer.Start(ctx, "repository.inbox.claim")
	defer span.End()

	rows, err := r.queries.ClaimInboxRecord(ctx, sqlc.ClaimInboxRecordParams{
		ID:        id,
		LockOwner: agentdb.PgText(owner),
		Column3:   agentdb.PgText(fmt.Sprintf("%d", ttlSeconds)),
	})
	if err != nil {
		span.RecordError(err)
		return false, err
	}

	return rows > 0, nil
}

func (r *inboxRepoImpl) Complete(ctx context.Context, id int64) (bool, error) {
	ctx, span := inboxRepoTracer.Start(ctx, "repository.inbox.complete")
	defer span.End()

	rows, err := r.queries.CompleteInboxRecord(ctx, id)
	if err != nil {
		span.RecordError(err)
		return false, err
	}

	return rows > 0, nil
}

func (r *inboxRepoImpl) MarkFailed(ctx context.Context, id int64, errMsg string) error {
	ctx, span := inboxRepoTracer.Start(ctx, "repository.inbox.mark_failed")
	defer span.End()

	err := r.queries.MarkInboxRecordFailed(ctx, sqlc.MarkInboxRecordFailedParams{
		ID:        id,
		LastError: agentdb.PgText(errMsg),
	})
	if err != nil {
		span.RecordError(err)
		return err
	}

	return nil
}

func (r *inboxRepoImpl) MarkDiscarded(ctx context.Context, id int64, reason string) error {
	ctx, span := inboxRepoTracer.Start(ctx, "repository.inbox.mark_discarded")
	defer span.End()

	err := r.queries.MarkInboxRecordDiscarded(ctx, sqlc.MarkInboxRecordDiscardedParams{
		ID:        id,
		LastError: agentdb.PgText(reason),
	})
	if err != nil {
		span.RecordError(err)
		return err
	}

	return nil
}
