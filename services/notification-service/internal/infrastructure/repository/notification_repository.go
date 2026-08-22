package repository

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/open-mrp/api/services/notification-service/internal/domain"
	"github.com/open-mrp/api/services/notification-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/tracing"
)

var notificationRepoTracer = tracing.GetTracer("notification-service.notification_repository")

type notificationRepoImpl struct {
	db *sqlc.Queries
}

func NewNotificationRepo(db *sqlc.Queries) domain.NotificationRepo {
	return &notificationRepoImpl{db: db}
}

func (r *notificationRepoImpl) Create(ctx context.Context, n *domain.Notification) *apierror.APIError {
	ctx, span := notificationRepoTracer.Start(ctx, "repository.notification.create")
	defer span.End()

	if err := r.create(ctx, n); err != nil {
		if db.IsDuplicateEntry(err) {
			return nil // idempotent — notification already exists
		}
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
	}
	return nil
}

func (r *notificationRepoImpl) CreateBatch(ctx context.Context, notifications []*domain.Notification) *apierror.APIError {
	ctx, span := notificationRepoTracer.Start(ctx, "repository.notification.create_batch")
	defer span.End()

	for _, n := range notifications {
		if err := r.create(ctx, n); err != nil {
			if db.IsDuplicateEntry(err) {
				continue // idempotent — skip rows that already exist
			}
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return tracing.Trace(span, apiErr)
			}
		}
	}
	return nil
}

func (r *notificationRepoImpl) create(ctx context.Context, n *domain.Notification) error {
	priority := n.Priority
	if priority == "" {
		priority = "normal"
	}
	return r.db.CreateNotification(ctx, sqlc.CreateNotificationParams{
		ID:                     n.ID,
		AccountID:              n.AccountID,
		RecipientAccountUserID: n.RecipientAccountUserID,
		Category:               n.Category,
		SourceMessageID:        db.NullStringPtr(n.SourceMessageID),
		ConversationID:         db.NullStringPtr(n.ConversationID),
		Title:                  n.Title,
		Body:                   db.NullStringPtr(n.Body),
		TemplateKey:            db.NullStringPtr(n.TemplateKey),
		TemplateParams:         db.NullableRawMessage(n.TemplateParams),
		LinkResourceType:       db.NullStringPtr(n.LinkResourceType),
		LinkResourceID:         db.NullStringPtr(n.LinkResourceID),
		SenderType:             db.NullStringPtr(n.SenderType),
		SenderID:               db.NullStringPtr(n.SenderID),
		SenderName:             db.NullStringPtr(n.SenderName),
		Priority:               priority,
		SeenAt:                 db.NullTimePtr(n.SeenAt),
		ReadAt:                 db.NullTimePtr(n.ReadAt),
		DismissedAt:            db.NullTimePtr(n.DismissedAt),
		Metadata:               db.NullableRawMessage(n.Metadata),
	})
}

func (r *notificationRepoImpl) GetByID(ctx context.Context, id, recipientAccountUserID string) (*domain.Notification, *apierror.APIError) {
	ctx, span := notificationRepoTracer.Start(ctx, "repository.notification.get_by_id")
	defer span.End()

	row, err := r.db.GetNotificationByID(ctx, sqlc.GetNotificationByIDParams{
		ID:                     id,
		RecipientAccountUserID: recipientAccountUserID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			return nil, apiErr
		}
		return nil, tracing.Trace(span, apiErr)
	}

	return toDomainNotification(row), nil
}

func (r *notificationRepoImpl) List(ctx context.Context, filter domain.NotificationListFilter) ([]*domain.Notification, *apierror.APIError) {
	ctx, span := notificationRepoTracer.Start(ctx, "repository.notification.list")
	defer span.End()

	var status any
	if filter.Status != nil {
		status = *filter.Status
	}

	var search sql.NullString
	if filter.Search != nil && *filter.Search != "" {
		search = sql.NullString{String: "%" + *filter.Search + "%", Valid: true}
	}

	senderIDs := toNullStrings(filter.SenderIDs)
	senderTypes := toNullStrings(filter.SenderTypes)

	rows, err := r.db.ListNotifications(ctx, sqlc.ListNotificationsParams{
		RecipientAccountUserID:  filter.RecipientAccountUserID,
		Category:                db.NullStringPtr(filter.Category),
		Status:                  status,
		Search:                  search,
		IncludeSenderFilter:     boolToInt(len(senderIDs) > 0),
		SenderIds:               senderIDs,
		IncludeSenderTypeFilter: boolToInt(len(senderTypes) > 0),
		SenderTypes:             senderTypes,
		CursorCreatedAt:         db.NullTimePtr(filter.CursorCreatedAt),
		CursorID:                db.NullStringPtr(filter.CursorID),
		Limit:                   filter.Limit,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	notifications := make([]*domain.Notification, 0, len(rows))
	for _, row := range rows {
		notifications = append(notifications, toDomainNotification(row))
	}
	return notifications, nil
}

func (r *notificationRepoImpl) CountUnseen(ctx context.Context, recipientAccountUserID string) (int64, *apierror.APIError) {
	ctx, span := notificationRepoTracer.Start(ctx, "repository.notification.count_unseen")
	defer span.End()

	count, err := r.db.CountUnseenNotifications(ctx, recipientAccountUserID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return 0, tracing.Trace(span, apiErr)
	}
	return count, nil
}

func (r *notificationRepoImpl) MarkSeen(ctx context.Context, id, recipientAccountUserID string) (*domain.Notification, *apierror.APIError) {
	ctx, span := notificationRepoTracer.Start(ctx, "repository.notification.mark_seen")
	defer span.End()

	err := r.db.MarkNotificationSeen(ctx, sqlc.MarkNotificationSeenParams{ID: id, RecipientAccountUserID: recipientAccountUserID})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return r.GetByID(ctx, id, recipientAccountUserID)
}

func (r *notificationRepoImpl) MarkRead(ctx context.Context, id, recipientAccountUserID string) (*domain.Notification, *apierror.APIError) {
	ctx, span := notificationRepoTracer.Start(ctx, "repository.notification.mark_read")
	defer span.End()

	err := r.db.MarkNotificationRead(ctx, sqlc.MarkNotificationReadParams{ID: id, RecipientAccountUserID: recipientAccountUserID})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return r.GetByID(ctx, id, recipientAccountUserID)
}

func (r *notificationRepoImpl) MarkDismissed(ctx context.Context, id, recipientAccountUserID string) (*domain.Notification, *apierror.APIError) {
	ctx, span := notificationRepoTracer.Start(ctx, "repository.notification.mark_dismissed")
	defer span.End()

	err := r.db.MarkNotificationDismissed(ctx, sqlc.MarkNotificationDismissedParams{ID: id, RecipientAccountUserID: recipientAccountUserID})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return r.GetByID(ctx, id, recipientAccountUserID)
}

func (r *notificationRepoImpl) DismissBySourceMessage(ctx context.Context, accountID, sourceMessageID string) *apierror.APIError {
	ctx, span := notificationRepoTracer.Start(ctx, "repository.notification.dismiss_by_source_message")
	defer span.End()

	err := r.db.DismissNotificationsBySourceMessage(ctx, sqlc.DismissNotificationsBySourceMessageParams{
		AccountID:       accountID,
		SourceMessageID: db.NullString(sourceMessageID),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *notificationRepoImpl) MarkAllSeen(ctx context.Context, recipientAccountUserID string) (int64, *apierror.APIError) {
	ctx, span := notificationRepoTracer.Start(ctx, "repository.notification.mark_all_seen")
	defer span.End()

	count, err := r.db.MarkAllNotificationsSeen(ctx, recipientAccountUserID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return 0, tracing.Trace(span, apiErr)
	}
	return count, nil
}

func (r *notificationRepoImpl) DismissByConversation(ctx context.Context, recipientAccountUserID, conversationID string) (int64, *apierror.APIError) {
	ctx, span := notificationRepoTracer.Start(ctx, "repository.notification.dismiss_by_conversation")
	defer span.End()

	count, err := r.db.DismissNotificationsByConversation(ctx, sqlc.DismissNotificationsByConversationParams{
		RecipientAccountUserID: recipientAccountUserID,
		ConversationID:         db.NullString(conversationID),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return 0, tracing.Trace(span, apiErr)
	}
	return count, nil
}

func (r *notificationRepoImpl) ResolveAccountUserID(ctx context.Context, userID, accountID string) (string, *apierror.APIError) {
	ctx, span := notificationRepoTracer.Start(ctx, "repository.notification.resolve_account_user_id")
	defer span.End()

	id, err := r.db.GetAccountUserIDByUserAndAccount(ctx, sqlc.GetAccountUserIDByUserAndAccountParams{
		UserID:    userID,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			return "", apiErr
		}
		return "", tracing.Trace(span, apiErr)
	}
	return id, nil
}

func (r *notificationRepoImpl) ResolveUserID(ctx context.Context, accountUserID string) (string, *apierror.APIError) {
	ctx, span := notificationRepoTracer.Start(ctx, "repository.notification.resolve_user_id")
	defer span.End()

	userID, err := r.db.GetUserIDByAccountUserID(ctx, accountUserID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			return "", apiErr
		}
		return "", tracing.Trace(span, apiErr)
	}
	return userID, nil
}

func (r *notificationRepoImpl) ResolveRecipientContact(ctx context.Context, accountUserID string) (*domain.RecipientContact, *apierror.APIError) {
	ctx, span := notificationRepoTracer.Start(ctx, "repository.notification.resolve_recipient_contact")
	defer span.End()

	row, err := r.db.GetUserContactByAccountUserID(ctx, accountUserID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			return nil, apiErr
		}
		return nil, tracing.Trace(span, apiErr)
	}
	return &domain.RecipientContact{
		Email: row.Email.String,
		Name:  row.Name.String,
	}, nil
}

func (r *notificationRepoImpl) ListMessagingContacts(ctx context.Context, accountID, query string) ([]*domain.MessagingContact, *apierror.APIError) {
	ctx, span := notificationRepoTracer.Start(ctx, "repository.notification.list_messaging_contacts")
	defer span.End()

	rows, err := r.db.ListMessagingContacts(ctx, sqlc.ListMessagingContactsParams{
		AccountID: accountID,
		Name:      sql.NullString{String: "%" + query + "%", Valid: true},
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	contacts := make([]*domain.MessagingContact, 0, len(rows))
	for _, row := range rows {
		contacts = append(contacts, &domain.MessagingContact{
			Type:          domain.MessagingContactTypeUser,
			AccountUserID: row.AccountUserID,
			Name:          row.Name.String,
		})
	}
	return contacts, nil
}

func (r *notificationRepoImpl) CountUnseenByUserAccounts(ctx context.Context, userID string) ([]domain.AccountUnread, *apierror.APIError) {
	ctx, span := notificationRepoTracer.Start(ctx, "repository.notification.count_unseen_by_user_accounts")
	defer span.End()

	rows, err := r.db.CountUnseenNotificationsByUserAccounts(ctx, userID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	counts := make([]domain.AccountUnread, 0, len(rows))
	for _, row := range rows {
		counts = append(counts, domain.AccountUnread{AccountID: row.AccountID, Unread: row.Unread})
	}
	return counts, nil
}

func toDomainNotification(row sqlc.Notification) *domain.Notification {
	return &domain.Notification{
		ID:                     row.ID,
		AccountID:              row.AccountID,
		RecipientAccountUserID: row.RecipientAccountUserID,
		Category:               row.Category,
		SourceMessageID:        db.StringFromNullString(row.SourceMessageID),
		ConversationID:         db.StringFromNullString(row.ConversationID),
		Title:                  row.Title,
		Body:                   db.StringFromNullString(row.Body),
		TemplateKey:            db.StringFromNullString(row.TemplateKey),
		TemplateParams:         json.RawMessage(row.TemplateParams),
		LinkResourceType:       db.StringFromNullString(row.LinkResourceType),
		LinkResourceID:         db.StringFromNullString(row.LinkResourceID),
		SenderType:             db.StringFromNullString(row.SenderType),
		SenderID:               db.StringFromNullString(row.SenderID),
		SenderName:             db.StringFromNullString(row.SenderName),
		Priority:               row.Priority,
		SeenAt:                 db.TimeFromNullTime(row.SeenAt),
		ReadAt:                 db.TimeFromNullTime(row.ReadAt),
		DismissedAt:            db.TimeFromNullTime(row.DismissedAt),
		Metadata:               json.RawMessage(row.Metadata),
		CreatedAt:              row.CreatedAt,
		UpdatedAt:              row.UpdatedAt,
	}
}

// toNullStrings converts a string slice to a []sql.NullString for sqlc IN-slice params.
func toNullStrings(values []string) []sql.NullString {
	if len(values) == 0 {
		return nil
	}
	out := make([]sql.NullString, 0, len(values))
	for _, v := range values {
		out = append(out, sql.NullString{String: v, Valid: true})
	}
	return out
}

// boolToInt maps a filter-enabled flag to the 1/0 sentinel the ListNotifications query uses to short-circuit its IN clauses.
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
