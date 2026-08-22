package repository

import (
	"context"

	"github.com/open-mrp/api/services/notification-service/internal/domain"
	"github.com/open-mrp/api/services/notification-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/tracing"
)

var conversationLinkRepoTracer = tracing.GetTracer("notification-service.conversation_link_repository")

type conversationLinkRepoImpl struct {
	db *sqlc.Queries
}

func NewConversationLinkRepo(db *sqlc.Queries) domain.ConversationLinkRepo {
	return &conversationLinkRepoImpl{db: db}
}

func (r *conversationLinkRepoImpl) Create(ctx context.Context, l *domain.ConversationLink) *apierror.APIError {
	ctx, span := conversationLinkRepoTracer.Start(ctx, "repository.conversation_link.create")
	defer span.End()

	err := r.db.CreateConversationLink(ctx, sqlc.CreateConversationLinkParams{
		ID:                     l.ID,
		AccountID:              l.AccountID,
		ConversationID:         l.ConversationID,
		ResourceType:           l.ResourceType,
		ResourceID:             l.ResourceID,
		CreatedByParticipantID: db.NullStringPtr(l.CreatedByParticipantID),
	})
	if err != nil {
		if db.IsDuplicateEntry(err) {
			return tracing.Trace(span, apierror.NewResourceExistsError("That record is already linked to this conversation."))
		}
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
	}
	return nil
}

func (r *conversationLinkRepoImpl) Delete(ctx context.Context, linkID, conversationID, accountID string) (bool, *apierror.APIError) {
	ctx, span := conversationLinkRepoTracer.Start(ctx, "repository.conversation_link.delete")
	defer span.End()

	rows, err := r.db.DeleteConversationLink(ctx, sqlc.DeleteConversationLinkParams{
		ID:             linkID,
		ConversationID: conversationID,
		AccountID:      accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}
	return rows > 0, nil
}

func (r *conversationLinkRepoImpl) List(ctx context.Context, conversationID, accountID string) ([]*domain.ConversationLink, *apierror.APIError) {
	ctx, span := conversationLinkRepoTracer.Start(ctx, "repository.conversation_link.list")
	defer span.End()

	rows, err := r.db.ListConversationLinks(ctx, sqlc.ListConversationLinksParams{
		ConversationID: conversationID,
		AccountID:      accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	out := make([]*domain.ConversationLink, 0, len(rows))
	for _, row := range rows {
		out = append(out, &domain.ConversationLink{
			ID:                     row.ID,
			AccountID:              row.AccountID,
			ConversationID:         row.ConversationID,
			ResourceType:           row.ResourceType,
			ResourceID:             row.ResourceID,
			CreatedByParticipantID: db.StringFromNullString(row.CreatedByParticipantID),
			CreatedAt:              row.CreatedAt,
		})
	}
	return out, nil
}
