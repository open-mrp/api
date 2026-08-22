package repository

import (
	"context"
	"encoding/json"

	"github.com/open-mrp/api/services/notification-service/internal/domain"
	"github.com/open-mrp/api/services/notification-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/id"
	"github.com/open-mrp/api/shared/tracing"
)

var announcementRepoTracer = tracing.GetTracer("notification-service.announcement_repository")

type announcementRepoImpl struct {
	db *sqlc.Queries
}

func NewAnnouncementRepo(db *sqlc.Queries) domain.AnnouncementRepo {
	return &announcementRepoImpl{db: db}
}

func (r *announcementRepoImpl) Create(ctx context.Context, announcementID string, input *domain.CreateAnnouncementInput) *apierror.APIError {
	ctx, span := announcementRepoTracer.Start(ctx, "repository.announcement.create")
	defer span.End()

	priority := input.Priority
	if priority == "" {
		priority = "normal"
	}
	err := r.db.CreateAnnouncement(ctx, sqlc.CreateAnnouncementParams{
		ID:               announcementID,
		Scope:            input.Scope,
		AccountID:        db.NullStringPtr(input.AccountID),
		Category:         input.Category,
		TemplateKey:      db.NullStringPtr(input.TemplateKey),
		TemplateParams:   db.NullableRawMessage(input.TemplateParams),
		Title:            input.Title,
		Body:             db.NullStringPtr(input.Body),
		LinkResourceType: db.NullStringPtr(input.LinkResourceType),
		LinkResourceID:   db.NullStringPtr(input.LinkResourceID),
		Priority:         priority,
		Audience:         nil,
		PublishAt:        input.PublishAt,
		ExpiresAt:        db.NullTimePtr(input.ExpiresAt),
		CreatedBy:        db.NullStringPtr(input.CreatedBy),
	})
	if err != nil {
		if db.IsDuplicateEntry(err) {
			return nil // idempotent — announcement already exists
		}
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
	}
	return nil
}

func (r *announcementRepoImpl) GetActiveByID(ctx context.Context, announcementID, accountUserID string, accountID *string) (*domain.Announcement, *apierror.APIError) {
	ctx, span := announcementRepoTracer.Start(ctx, "repository.announcement.get_active_by_id")
	defer span.End()

	row, err := r.db.GetActiveAnnouncementByID(ctx, sqlc.GetActiveAnnouncementByIDParams{
		AccountUserID: accountUserID,
		ID:            announcementID,
		AccountID:     db.NullStringPtr(accountID),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			return nil, apiErr
		}
		return nil, tracing.Trace(span, apiErr)
	}
	return getRowToDomainAnnouncement(row), nil
}

func (r *announcementRepoImpl) ListActive(ctx context.Context, filter domain.AnnouncementListFilter) ([]*domain.Announcement, *apierror.APIError) {
	ctx, span := announcementRepoTracer.Start(ctx, "repository.announcement.list_active")
	defer span.End()

	rows, err := r.db.ListActiveAnnouncements(ctx, sqlc.ListActiveAnnouncementsParams{
		AccountUserID:   filter.AccountUserID,
		AccountID:       db.NullStringPtr(filter.AccountID),
		CursorPublishAt: db.NullTimePtr(filter.CursorPublishAt),
		CursorID:        db.NullStringPtr(filter.CursorID),
		Limit:           filter.Limit,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	announcements := make([]*domain.Announcement, 0, len(rows))
	for _, row := range rows {
		announcements = append(announcements, listRowToDomainAnnouncement(row))
	}
	return announcements, nil
}

func (r *announcementRepoImpl) CountUnseen(ctx context.Context, accountUserID string, accountID *string) (int64, *apierror.APIError) {
	ctx, span := announcementRepoTracer.Start(ctx, "repository.announcement.count_unseen")
	defer span.End()

	count, err := r.db.CountUnseenAnnouncements(ctx, sqlc.CountUnseenAnnouncementsParams{
		AccountUserID: accountUserID,
		AccountID:     db.NullStringPtr(accountID),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return 0, tracing.Trace(span, apiErr)
	}
	return count, nil
}

func (r *announcementRepoImpl) MarkSeen(ctx context.Context, announcementID, accountUserID string) *apierror.APIError {
	ctx, span := announcementRepoTracer.Start(ctx, "repository.announcement.mark_seen")
	defer span.End()

	receiptID, apiErr := r.newReceiptID()
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	err := r.db.UpsertAnnouncementSeen(ctx, sqlc.UpsertAnnouncementSeenParams{
		ID:             receiptID,
		AnnouncementID: announcementID,
		AccountUserID:  accountUserID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *announcementRepoImpl) MarkRead(ctx context.Context, announcementID, accountUserID string) *apierror.APIError {
	ctx, span := announcementRepoTracer.Start(ctx, "repository.announcement.mark_read")
	defer span.End()

	receiptID, apiErr := r.newReceiptID()
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	err := r.db.UpsertAnnouncementRead(ctx, sqlc.UpsertAnnouncementReadParams{
		ID:             receiptID,
		AnnouncementID: announcementID,
		AccountUserID:  accountUserID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *announcementRepoImpl) MarkDismissed(ctx context.Context, announcementID, accountUserID string) *apierror.APIError {
	ctx, span := announcementRepoTracer.Start(ctx, "repository.announcement.mark_dismissed")
	defer span.End()

	receiptID, apiErr := r.newReceiptID()
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	err := r.db.UpsertAnnouncementDismissed(ctx, sqlc.UpsertAnnouncementDismissedParams{
		ID:             receiptID,
		AnnouncementID: announcementID,
		AccountUserID:  accountUserID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

// newReceiptID generates an id for a lazily-created announcement receipt. The unique (announcement_id, account_user_id) key dedupes concurrent inserts, so a fresh id per call is fine — losers collapse into the ON DUPLICATE KEY UPDATE path.
func (r *announcementRepoImpl) newReceiptID() (string, *apierror.APIError) {
	return id.GenID(id.AnnouncementReceiptIDPrefix, nil)
}

func (r *announcementRepoImpl) CountUnseenByUserAccounts(ctx context.Context, userID string) ([]domain.AccountUnread, *apierror.APIError) {
	ctx, span := announcementRepoTracer.Start(ctx, "repository.announcement.count_unseen_by_user_accounts")
	defer span.End()

	rows, err := r.db.CountUnseenAnnouncementsByUserAccounts(ctx, userID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	counts := make([]domain.AccountUnread, 0, len(rows))
	for _, row := range rows {
		counts = append(counts, domain.AccountUnread{AccountID: row.AccountID, Unread: row.Unread})
	}
	return counts, nil
}

func getRowToDomainAnnouncement(row sqlc.GetActiveAnnouncementByIDRow) *domain.Announcement {
	return &domain.Announcement{
		ID:               row.ID,
		Scope:            row.Scope,
		AccountID:        db.StringFromNullString(row.AccountID),
		Category:         row.Category,
		TemplateKey:      db.StringFromNullString(row.TemplateKey),
		TemplateParams:   json.RawMessage(row.TemplateParams),
		Title:            row.Title,
		Body:             db.StringFromNullString(row.Body),
		LinkResourceType: db.StringFromNullString(row.LinkResourceType),
		LinkResourceID:   db.StringFromNullString(row.LinkResourceID),
		Priority:         row.Priority,
		PublishAt:        row.PublishAt,
		ExpiresAt:        db.TimeFromNullTime(row.ExpiresAt),
		CreatedBy:        db.StringFromNullString(row.CreatedBy),
		CreatedAt:        row.CreatedAt,
		UpdatedAt:        row.UpdatedAt,
		SeenAt:           db.TimeFromNullTime(row.ReceiptSeenAt),
		ReadAt:           db.TimeFromNullTime(row.ReceiptReadAt),
		DismissedAt:      db.TimeFromNullTime(row.ReceiptDismissedAt),
	}
}

func listRowToDomainAnnouncement(row sqlc.ListActiveAnnouncementsRow) *domain.Announcement {
	return &domain.Announcement{
		ID:               row.ID,
		Scope:            row.Scope,
		AccountID:        db.StringFromNullString(row.AccountID),
		Category:         row.Category,
		TemplateKey:      db.StringFromNullString(row.TemplateKey),
		TemplateParams:   json.RawMessage(row.TemplateParams),
		Title:            row.Title,
		Body:             db.StringFromNullString(row.Body),
		LinkResourceType: db.StringFromNullString(row.LinkResourceType),
		LinkResourceID:   db.StringFromNullString(row.LinkResourceID),
		Priority:         row.Priority,
		PublishAt:        row.PublishAt,
		ExpiresAt:        db.TimeFromNullTime(row.ExpiresAt),
		CreatedBy:        db.StringFromNullString(row.CreatedBy),
		CreatedAt:        row.CreatedAt,
		UpdatedAt:        row.UpdatedAt,
		SeenAt:           db.TimeFromNullTime(row.ReceiptSeenAt),
		ReadAt:           db.TimeFromNullTime(row.ReceiptReadAt),
		DismissedAt:      db.TimeFromNullTime(row.ReceiptDismissedAt),
	}
}
