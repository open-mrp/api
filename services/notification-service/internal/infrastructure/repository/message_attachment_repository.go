package repository

import (
	"context"

	"github.com/open-mrp/api/services/notification-service/internal/domain"
	"github.com/open-mrp/api/services/notification-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/tracing"
)

var messageAttachmentRepoTracer = tracing.GetTracer("notification-service.message_attachment_repository")

type messageAttachmentRepoImpl struct {
	db *sqlc.Queries
}

func NewMessageAttachmentRepo(db *sqlc.Queries) domain.MessageAttachmentRepo {
	return &messageAttachmentRepoImpl{db: db}
}

func (r *messageAttachmentRepoImpl) Create(ctx context.Context, a *domain.MessageAttachment) *apierror.APIError {
	ctx, span := messageAttachmentRepoTracer.Start(ctx, "repository.message_attachment.create")
	defer span.End()
	err := r.db.CreateMessageAttachment(ctx, sqlc.CreateMessageAttachmentParams{
		ID:           a.ID,
		MessageID:    a.MessageID,
		AccountID:    a.AccountID,
		Kind:         a.Kind,
		S3Key:        db.NullStringPtr(a.S3Key),
		Url:          db.NullStringPtr(a.URL),
		Filename:     db.NullStringPtr(a.Filename),
		ContentType:  db.NullStringPtr(a.ContentType),
		SizeBytes:    db.NullInt64Ptr(a.SizeBytes),
		ResourceType: db.NullStringPtr(a.ResourceType),
		ResourceID:   db.NullStringPtr(a.ResourceID),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *messageAttachmentRepoImpl) ListByMessageIDs(ctx context.Context, messageIDs []string) ([]*domain.MessageAttachment, *apierror.APIError) {
	ctx, span := messageAttachmentRepoTracer.Start(ctx, "repository.message_attachment.list_by_message_ids")
	defer span.End()
	if len(messageIDs) == 0 {
		return nil, nil
	}
	rows, err := r.db.ListMessageAttachmentsByMessageIDs(ctx, messageIDs)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	attachments := make([]*domain.MessageAttachment, 0, len(rows))
	for _, row := range rows {
		attachments = append(attachments, messageAttachmentFromRow(row))
	}
	return attachments, nil
}

func (r *messageAttachmentRepoImpl) ListByConversation(ctx context.Context, conversationID string) ([]*domain.MessageAttachment, *apierror.APIError) {
	ctx, span := messageAttachmentRepoTracer.Start(ctx, "repository.message_attachment.list_by_conversation")
	defer span.End()
	rows, err := r.db.ListConversationAttachments(ctx, conversationID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	attachments := make([]*domain.MessageAttachment, 0, len(rows))
	for _, row := range rows {
		attachments = append(attachments, &domain.MessageAttachment{
			ID:    row.ID,
			S3Key: db.StringFromNullString(row.S3Key),
		})
	}
	return attachments, nil
}

func (r *messageAttachmentRepoImpl) DeleteByID(ctx context.Context, id string) *apierror.APIError {
	ctx, span := messageAttachmentRepoTracer.Start(ctx, "repository.message_attachment.delete_by_id")
	defer span.End()
	if err := r.db.DeleteMessageAttachmentByID(ctx, id); err != nil {
		return tracing.Trace(span, db.MapSQLError(err))
	}
	return nil
}

func messageAttachmentFromRow(row sqlc.MessageAttachment) *domain.MessageAttachment {
	a := &domain.MessageAttachment{
		ID:           row.ID,
		MessageID:    row.MessageID,
		AccountID:    row.AccountID,
		Kind:         row.Kind,
		S3Key:        db.StringFromNullString(row.S3Key),
		URL:          db.StringFromNullString(row.Url),
		Filename:     db.StringFromNullString(row.Filename),
		ContentType:  db.StringFromNullString(row.ContentType),
		ResourceType: db.StringFromNullString(row.ResourceType),
		ResourceID:   db.StringFromNullString(row.ResourceID),
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
	}
	if row.SizeBytes.Valid {
		size := row.SizeBytes.Int64
		a.SizeBytes = &size
	}
	return a
}
