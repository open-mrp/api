package repository

import (
	"context"
	"fmt"

	agentdb "github.com/augno/api/services/agent-service/internal/infrastructure/db"
	"github.com/augno/api/services/agent-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/messaging"
	"github.com/augno/api/shared/tracing"
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
	}

	return record, nil
}

func (r *inboxRepoImpl) MarkProcessed(ctx context.Context, id int64) error {
	ctx, span := inboxRepoTracer.Start(ctx, "repository.inbox.mark_processed")
	defer span.End()

	err := r.queries.MarkInboxRecordProcessed(ctx, id)
	if err != nil {
		span.RecordError(err)
		return err
	}

	return nil
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
