package repository

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/open-mrp/api/services/notification-service/internal/domain"
	"github.com/open-mrp/api/services/notification-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/tracing"
)

var messageRepoTracer = tracing.GetTracer("notification-service.message_repository")

type messageRepoImpl struct {
	db *sqlc.Queries
}

func NewMessageRepo(db *sqlc.Queries) domain.MessageRepo {
	return &messageRepoImpl{db: db}
}

func (r *messageRepoImpl) Create(ctx context.Context, m *domain.Message) (bool, *apierror.APIError) {
	ctx, span := messageRepoTracer.Start(ctx, "repository.message.create")
	defer span.End()

	kind := m.Kind
	if kind == "" {
		kind = "chat"
	}
	// streaming_state is NOT NULL (default 'complete'); since the column is in the INSERT list we must supply a value rather than relying on the DB default.
	streamingState := "complete"
	if m.StreamingState != nil && *m.StreamingState != "" {
		streamingState = *m.StreamingState
	}
	// visibility is NOT NULL (default 'internal'); since the column is in the INSERT list we must supply a value rather than relying on the DB default.
	visibility := m.Visibility
	if visibility == "" {
		visibility = "internal"
	}
	channel := resolveCreateChannel(m)
	err := r.db.CreateMessage(ctx, sqlc.CreateMessageParams{
		ID:                  m.ID,
		ConversationID:      m.ConversationID,
		AccountID:           m.AccountID,
		Sequence:            sql.NullInt64{Int64: m.Sequence, Valid: true},
		Kind:                kind,
		Visibility:          visibility,
		Channel:             db.NullStringPtr(channel),
		SenderParticipantID: db.NullStringPtr(m.SenderParticipantID),
		ClientMessageID:     db.NullStringPtr(m.ClientMessageID),
		Body:                db.NullStringPtr(m.Body),
		Preview:             db.NullStringPtr(m.Preview),
		Subject:             db.NullStringPtr(m.Subject),
		EventType:           db.NullStringPtr(m.EventType),
		LinkResourceType:    db.NullStringPtr(m.LinkResourceType),
		LinkResourceID:      db.NullStringPtr(m.LinkResourceID),
		AgentRunID:          db.NullStringPtr(m.AgentRunID),
		ReplyToMessageID:    db.NullStringPtr(m.ReplyToMessageID),
		StreamingState:      streamingState,
		Metadata:            db.NullableRawMessage(m.Metadata),
	})
	if err != nil {
		if db.IsDuplicateEntry(err) {
			return false, nil // idempotent — caller resolves the existing row via GetByClientID
		}
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return false, tracing.Trace(span, apiErr)
		}
	}
	return true, nil
}

func (r *messageRepoImpl) GetByID(ctx context.Context, id, accountID string) (*domain.Message, *apierror.APIError) {
	ctx, span := messageRepoTracer.Start(ctx, "repository.message.get_by_id")
	defer span.End()

	row, err := r.db.GetMessageByID(ctx, sqlc.GetMessageByIDParams{ID: id, AccountID: accountID})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			return nil, apiErr
		}
		return nil, tracing.Trace(span, apiErr)
	}
	return messageFromRow(row), nil
}

func (r *messageRepoImpl) GetByIDs(ctx context.Context, ids []string) ([]*domain.Message, *apierror.APIError) {
	ctx, span := messageRepoTracer.Start(ctx, "repository.message.get_by_ids")
	defer span.End()

	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := r.db.GetMessagesByIDs(ctx, ids)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	messages := make([]*domain.Message, 0, len(rows))
	for _, row := range rows {
		messages = append(messages, messageFromRow(row))
	}
	return messages, nil
}

func (r *messageRepoImpl) GetByClientID(ctx context.Context, conversationID, clientMessageID string) (*domain.Message, *apierror.APIError) {
	ctx, span := messageRepoTracer.Start(ctx, "repository.message.get_by_client_id")
	defer span.End()

	row, err := r.db.GetMessageByClientID(ctx, sqlc.GetMessageByClientIDParams{
		ConversationID:  conversationID,
		ClientMessageID: db.NullStringPtr(&clientMessageID),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			return nil, apiErr
		}
		return nil, tracing.Trace(span, apiErr)
	}
	return messageFromRow(row), nil
}

func (r *messageRepoImpl) List(ctx context.Context, filter domain.MessageListFilter) ([]*domain.Message, *apierror.APIError) {
	ctx, span := messageRepoTracer.Start(ctx, "repository.message.list")
	defer span.End()

	// Customer-relation viewers (IncludeInternal=false) get the visibility-filtered query so an internal note is never serialized into a customer payload; staff get the full history.
	if !filter.IncludeInternal {
		rows, err := r.db.ListMessagesVisible(ctx, sqlc.ListMessagesVisibleParams{
			ConversationID: filter.ConversationID,
			BeforeSequence: db.NullInt64Ptr(filter.BeforeSequence),
			AfterSequence:  db.NullInt64Ptr(filter.AfterSequence),
			Limit:          filter.Limit,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		messages := make([]*domain.Message, 0, len(rows))
		for _, row := range rows {
			messages = append(messages, messageFromRow(row))
		}
		return messages, nil
	}

	rows, err := r.db.ListMessages(ctx, sqlc.ListMessagesParams{
		ConversationID: filter.ConversationID,
		BeforeSequence: db.NullInt64Ptr(filter.BeforeSequence),
		AfterSequence:  db.NullInt64Ptr(filter.AfterSequence),
		Limit:          filter.Limit,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	messages := make([]*domain.Message, 0, len(rows))
	for _, row := range rows {
		messages = append(messages, messageFromRow(row))
	}
	return messages, nil
}

// GetLastVisible returns the most recent customer-visible (non-internal, non-deleted) message in a conversation, for a customer viewer's last-message preview. Returns nil when none exist.
func (r *messageRepoImpl) GetLastVisible(ctx context.Context, conversationID string) (*domain.Message, *apierror.APIError) {
	ctx, span := messageRepoTracer.Start(ctx, "repository.message.get_last_visible")
	defer span.End()
	row, err := r.db.GetLastVisibleMessage(ctx, conversationID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			return nil, nil
		}
		return nil, tracing.Trace(span, apiErr)
	}
	return messageFromRow(row), nil
}

// CountVisibleAfter counts customer-visible messages after a read cursor, for a customer's unread badge.
func (r *messageRepoImpl) CountVisibleAfter(ctx context.Context, conversationID string, afterSequence int64) (int64, *apierror.APIError) {
	ctx, span := messageRepoTracer.Start(ctx, "repository.message.count_visible_after")
	defer span.End()
	n, err := r.db.CountVisibleMessagesAfter(ctx, sqlc.CountVisibleMessagesAfterParams{
		ConversationID: conversationID,
		AfterSequence:  sql.NullInt64{Int64: afterSequence, Valid: true},
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return 0, tracing.Trace(span, apiErr)
	}
	return n, nil
}

func (r *messageRepoImpl) UpdateBody(ctx context.Context, id, accountID string, body, preview *string) *apierror.APIError {
	ctx, span := messageRepoTracer.Start(ctx, "repository.message.update_body")
	defer span.End()
	err := r.db.UpdateMessageBody(ctx, sqlc.UpdateMessageBodyParams{
		Body:      db.NullStringPtr(body),
		Preview:   db.NullStringPtr(preview),
		ID:        id,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

// SetStreamingBody updates an in-flight agent reply's body/preview (and optionally flips its streaming_state to "complete"), without marking it edited. Returns whether a row was actually updated
// — false means the message was already complete/deleted or doesn't exist yet, so the caller should skip the realtime push. See SetMessageStreamingBody for the streaming_state guard.
func (r *messageRepoImpl) SetStreamingBody(ctx context.Context, id, accountID string, body, preview *string, state string) (bool, *apierror.APIError) {
	ctx, span := messageRepoTracer.Start(ctx, "repository.message.set_streaming_body")
	defer span.End()
	rows, err := r.db.SetMessageStreamingBody(ctx, sqlc.SetMessageStreamingBodyParams{
		Body:           db.NullStringPtr(body),
		Preview:        db.NullStringPtr(preview),
		StreamingState: state,
		ID:             id,
		AccountID:      accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}
	return rows > 0, nil
}

func (r *messageRepoImpl) SetMessageMetadata(ctx context.Context, id, accountID string, metadata json.RawMessage) *apierror.APIError {
	ctx, span := messageRepoTracer.Start(ctx, "repository.message.set_message_metadata")
	defer span.End()
	_, err := r.db.SetMessageMetadata(ctx, sqlc.SetMessageMetadataParams{
		Metadata:  db.NullableRawMessage(metadata),
		ID:        id,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *messageRepoImpl) SoftDelete(ctx context.Context, id, accountID string) *apierror.APIError {
	ctx, span := messageRepoTracer.Start(ctx, "repository.message.soft_delete")
	defer span.End()
	err := r.db.SoftDeleteMessage(ctx, sqlc.SoftDeleteMessageParams{ID: id, AccountID: accountID})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func messageFromRow(row sqlc.Message) *domain.Message {
	streamingState := row.StreamingState
	return &domain.Message{
		ID:                      row.ID,
		ConversationID:          row.ConversationID,
		AccountID:               row.AccountID,
		Sequence:                row.Sequence.Int64,
		Kind:                    row.Kind,
		Status:                  row.Status,
		Visibility:              row.Visibility,
		SenderParticipantID:     db.StringFromNullString(row.SenderParticipantID),
		ClientMessageID:         db.StringFromNullString(row.ClientMessageID),
		Body:                    db.StringFromNullString(row.Body),
		Subject:                 db.StringFromNullString(row.Subject),
		Channel:                 new(string(constants.ResolveMessageChannel(db.StringFromNullString(row.Channel), row.Kind))),
		SourceThreadMessageID:   db.StringFromNullString(row.SourceThreadMessageID),
		ApprovedByAccountUserID: db.StringFromNullString(row.ApprovedByAccountUserID),
		ScheduledFor:            db.TimeFromNullTime(row.ScheduledFor),
		ScheduledAttempts:       row.ScheduledAttempts,
		LastError:               db.StringFromNullString(row.LastError),
		LockedAt:                db.TimeFromNullTime(row.LockedAt),
		LockOwner:               db.StringFromNullString(row.LockOwner),
		Preview:                 db.StringFromNullString(row.Preview),
		EventType:               db.StringFromNullString(row.EventType),
		LinkResourceType:        db.StringFromNullString(row.LinkResourceType),
		LinkResourceID:          db.StringFromNullString(row.LinkResourceID),
		AgentRunID:              db.StringFromNullString(row.AgentRunID),
		ReplyToMessageID:        db.StringFromNullString(row.ReplyToMessageID),
		StreamingState:          &streamingState,
		EditedAt:                db.TimeFromNullTime(row.EditedAt),
		DeletedAt:               db.TimeFromNullTime(row.DeletedAt),
		Metadata:                json.RawMessage(row.Metadata),
		CreatedAt:               row.CreatedAt,
		UpdatedAt:               row.UpdatedAt,
	}
}

// --- Customer-reply drafts (status=draft message rows) ---

func (r *messageRepoImpl) CreateDraft(ctx context.Context, m *domain.Message) *apierror.APIError {
	ctx, span := messageRepoTracer.Start(ctx, "repository.message.create_draft")
	defer span.End()
	err := r.db.CreateDraftMessage(ctx, sqlc.CreateDraftMessageParams{
		ID:                    m.ID,
		ConversationID:        m.ConversationID,
		AccountID:             m.AccountID,
		Channel:               db.NullStringPtr(m.Channel),
		Subject:               db.NullStringPtr(m.Subject),
		SenderParticipantID:   db.NullStringPtr(m.SenderParticipantID),
		AgentRunID:            db.NullStringPtr(m.AgentRunID),
		SourceThreadMessageID: db.NullStringPtr(m.SourceThreadMessageID),
		Body:                  db.NullStringPtr(m.Body),
		Preview:               db.NullStringPtr(m.Preview),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *messageRepoImpl) ListDrafts(ctx context.Context, conversationID, accountID string, status *string) ([]*domain.Message, *apierror.APIError) {
	ctx, span := messageRepoTracer.Start(ctx, "repository.message.list_drafts")
	defer span.End()
	rows, err := r.db.ListDraftMessagesByConversation(ctx, sqlc.ListDraftMessagesByConversationParams{
		ConversationID: conversationID,
		AccountID:      accountID,
		Status:         db.NullStringPtr(status),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	out := make([]*domain.Message, 0, len(rows))
	for _, row := range rows {
		out = append(out, messageFromRow(row))
	}
	return out, nil
}

func (r *messageRepoImpl) UpdateDraftContent(ctx context.Context, id, accountID string, body string, subject, preview *string) *apierror.APIError {
	ctx, span := messageRepoTracer.Start(ctx, "repository.message.update_draft_content")
	defer span.End()
	err := r.db.UpdateDraftMessageContent(ctx, sqlc.UpdateDraftMessageContentParams{
		Body:      db.NullStringPtr(&body),
		Subject:   db.NullStringPtr(subject),
		Preview:   db.NullStringPtr(preview),
		ID:        id,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *messageRepoImpl) SetDraftStatus(ctx context.Context, id, accountID, status string) (bool, *apierror.APIError) {
	ctx, span := messageRepoTracer.Start(ctx, "repository.message.set_draft_status")
	defer span.End()
	rows, err := r.db.SetDraftMessageStatus(ctx, sqlc.SetDraftMessageStatusParams{
		Status:    status,
		ID:        id,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}
	return rows > 0, nil
}

func (r *messageRepoImpl) SupersedeDraftsForThread(ctx context.Context, conversationID, sourceThreadMessageID string) *apierror.APIError {
	ctx, span := messageRepoTracer.Start(ctx, "repository.message.supersede_drafts_for_thread")
	defer span.End()
	err := r.db.SupersedeDraftMessagesForThread(ctx, sqlc.SupersedeDraftMessagesForThreadParams{
		ConversationID:        conversationID,
		SourceThreadMessageID: db.NullStringPtr(&sourceThreadMessageID),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *messageRepoImpl) PromoteDraft(ctx context.Context, id, accountID, kind string, sequence int64, approvedByAccountUserID, preview *string) (bool, *apierror.APIError) {
	ctx, span := messageRepoTracer.Start(ctx, "repository.message.promote_draft")
	defer span.End()
	rows, err := r.db.PromoteDraftMessage(ctx, sqlc.PromoteDraftMessageParams{
		Sequence:                sql.NullInt64{Int64: sequence, Valid: true},
		Kind:                    kind,
		ApprovedByAccountUserID: db.NullStringPtr(approvedByAccountUserID),
		Preview:                 db.NullStringPtr(preview),
		ID:                      id,
		AccountID:               accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}
	return rows > 0, nil
}

// --- Scheduled messages (status=scheduled message rows) ---

func (r *messageRepoImpl) CreateScheduled(ctx context.Context, m *domain.Message) *apierror.APIError {
	ctx, span := messageRepoTracer.Start(ctx, "repository.message.create_scheduled")
	defer span.End()
	var scheduledFor sql.NullTime
	if m.ScheduledFor != nil {
		scheduledFor = sql.NullTime{Time: *m.ScheduledFor, Valid: true}
	}
	err := r.db.CreateScheduledMessage(ctx, sqlc.CreateScheduledMessageParams{
		ID:                  m.ID,
		ConversationID:      m.ConversationID,
		AccountID:           m.AccountID,
		SenderParticipantID: db.NullStringPtr(m.SenderParticipantID),
		Body:                db.NullStringPtr(m.Body),
		Preview:             db.NullStringPtr(m.Preview),
		Channel:             db.NullStringPtr(resolveCreateChannel(m)),
		ScheduledFor:        scheduledFor,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *messageRepoImpl) ListScheduledByConversation(ctx context.Context, conversationID, accountID, accountUserID string, limit int32) ([]*domain.Message, *apierror.APIError) {
	ctx, span := messageRepoTracer.Start(ctx, "repository.message.list_scheduled_by_conversation")
	defer span.End()
	rows, err := r.db.ListScheduledMessagesByConversation(ctx, sqlc.ListScheduledMessagesByConversationParams{
		AccountID:      accountID,
		ConversationID: conversationID,
		AccountUserID:  db.NullStringPtr(&accountUserID),
		Limit:          limit,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	out := make([]*domain.Message, 0, len(rows))
	for _, row := range rows {
		out = append(out, messageFromRow(row))
	}
	return out, nil
}

func (r *messageRepoImpl) CancelScheduled(ctx context.Context, id, accountID, accountUserID string) (bool, *apierror.APIError) {
	ctx, span := messageRepoTracer.Start(ctx, "repository.message.cancel_scheduled")
	defer span.End()
	rows, err := r.db.CancelScheduledMessageForUser(ctx, sqlc.CancelScheduledMessageForUserParams{
		ID:            id,
		AccountID:     accountID,
		AccountUserID: db.NullStringPtr(&accountUserID),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}
	return rows > 0, nil
}

func (r *messageRepoImpl) ListDueScheduled(ctx context.Context, limit int32) ([]*domain.Message, *apierror.APIError) {
	ctx, span := messageRepoTracer.Start(ctx, "repository.message.list_due_scheduled")
	defer span.End()
	rows, err := r.db.ListDueScheduledMessages(ctx, limit)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	out := make([]*domain.Message, 0, len(rows))
	for _, row := range rows {
		out = append(out, messageFromRow(row))
	}
	return out, nil
}

func (r *messageRepoImpl) ClaimScheduled(ctx context.Context, id, lockOwner string) (bool, *apierror.APIError) {
	ctx, span := messageRepoTracer.Start(ctx, "repository.message.claim_scheduled")
	defer span.End()
	rows, err := r.db.ClaimScheduledMessage(ctx, sqlc.ClaimScheduledMessageParams{
		LockOwner: db.NullStringPtr(&lockOwner),
		ID:        id,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}
	return rows > 0, nil
}

func (r *messageRepoImpl) PromoteScheduled(ctx context.Context, id string, sequence int64) (bool, *apierror.APIError) {
	ctx, span := messageRepoTracer.Start(ctx, "repository.message.promote_scheduled")
	defer span.End()
	rows, err := r.db.PromoteScheduledMessage(ctx, sqlc.PromoteScheduledMessageParams{
		Sequence: sql.NullInt64{Int64: sequence, Valid: true},
		ID:       id,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}
	return rows > 0, nil
}

func (r *messageRepoImpl) MarkScheduledFailed(ctx context.Context, id, status string, lastError *string) *apierror.APIError {
	ctx, span := messageRepoTracer.Start(ctx, "repository.message.mark_scheduled_failed")
	defer span.End()
	err := r.db.MarkScheduledMessageFailed(ctx, sqlc.MarkScheduledMessageFailedParams{
		Status:    status,
		LastError: db.NullStringPtr(lastError),
		ID:        id,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func resolveCreateChannel(m *domain.Message) *string {
	if m.Channel != nil && *m.Channel != "" {
		return new(string(constants.ResolveMessageChannel(m.Channel, m.Kind)))
	}
	return new(string(constants.ResolveMessageChannel(nil, m.Kind)))
}
