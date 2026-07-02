package repository

import (
	"context"

	"github.com/augno/api/services/notification-service/internal/domain"
	"github.com/augno/api/services/notification-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/tracing"
)

var notificationPreferenceRepoTracer = tracing.GetTracer("notification-service.notification_preference_repository")

type notificationPreferenceRepoImpl struct {
	db *sqlc.Queries
}

func NewNotificationPreferenceRepo(db *sqlc.Queries) domain.NotificationPreferenceRepo {
	return &notificationPreferenceRepoImpl{db: db}
}

func (r *notificationPreferenceRepoImpl) List(ctx context.Context, accountUserID string) ([]*domain.NotificationPreference, *apierror.APIError) {
	ctx, span := notificationPreferenceRepoTracer.Start(ctx, "repository.notification_preference.list")
	defer span.End()
	rows, err := r.db.ListNotificationPreferences(ctx, accountUserID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	prefs := make([]*domain.NotificationPreference, 0, len(rows))
	for _, row := range rows {
		prefs = append(prefs, notificationPreferenceFromRow(row))
	}
	return prefs, nil
}

func (r *notificationPreferenceRepoImpl) GetEffective(ctx context.Context, accountUserID, category string) (*domain.EffectiveNotificationPreference, *apierror.APIError) {
	ctx, span := notificationPreferenceRepoTracer.Start(ctx, "repository.notification_preference.get_effective")
	defer span.End()
	row, err := r.db.GetEffectiveNotificationPreference(ctx, sqlc.GetEffectiveNotificationPreferenceParams{
		AccountUserID: accountUserID,
		Category:      category,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			return nil, apiErr
		}
		return nil, tracing.Trace(span, apiErr)
	}
	return &domain.EffectiveNotificationPreference{
		InAppEnabled: row.InAppEnabled,
		EmailEnabled: row.EmailEnabled,
		PushEnabled:  row.PushEnabled,
		Digest:       row.Digest,
	}, nil
}

func (r *notificationPreferenceRepoImpl) Upsert(ctx context.Context, id, accountID, accountUserID string, input *domain.UpsertNotificationPreferenceInput) *apierror.APIError {
	ctx, span := notificationPreferenceRepoTracer.Start(ctx, "repository.notification_preference.upsert")
	defer span.End()
	err := r.db.UpsertNotificationPreference(ctx, sqlc.UpsertNotificationPreferenceParams{
		ID:            id,
		AccountID:     accountID,
		AccountUserID: accountUserID,
		Category:      input.Category,
		InAppEnabled:  input.InAppEnabled,
		EmailEnabled:  input.EmailEnabled,
		PushEnabled:   input.PushEnabled,
		Digest:        input.Digest,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *notificationPreferenceRepoImpl) GetByUserCategory(ctx context.Context, accountUserID, category string) (*domain.NotificationPreference, *apierror.APIError) {
	ctx, span := notificationPreferenceRepoTracer.Start(ctx, "repository.notification_preference.get_by_user_category")
	defer span.End()
	row, err := r.db.GetNotificationPreferenceByUserCategory(ctx, sqlc.GetNotificationPreferenceByUserCategoryParams{
		AccountUserID: accountUserID,
		Category:      category,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			return nil, apiErr
		}
		return nil, tracing.Trace(span, apiErr)
	}
	return notificationPreferenceFromRow(row), nil
}

func notificationPreferenceFromRow(row sqlc.NotificationPreference) *domain.NotificationPreference {
	return &domain.NotificationPreference{
		ID:            row.ID,
		AccountID:     row.AccountID,
		AccountUserID: row.AccountUserID,
		Category:      row.Category,
		InAppEnabled:  row.InAppEnabled,
		EmailEnabled:  row.EmailEnabled,
		PushEnabled:   row.PushEnabled,
		Digest:        row.Digest,
		CreatedAt:     row.CreatedAt,
		UpdatedAt:     row.UpdatedAt,
	}
}
