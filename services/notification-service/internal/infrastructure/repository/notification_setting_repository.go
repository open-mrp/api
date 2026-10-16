package repository

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/open-mrp/api/services/notification-service/internal/domain"
	"github.com/open-mrp/api/services/notification-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/id"
	"github.com/open-mrp/api/shared/notifyprefs"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/db"
	"github.com/open-mrp/apikit/tracing"
)

var notificationSettingRepoTracer = tracing.GetTracer("notification-service.notification_setting_repository")

type notificationSettingRepoImpl struct {
	db *sqlc.Queries
}

func NewNotificationSettingRepo(db *sqlc.Queries) domain.NotificationSettingRepo {
	return &notificationSettingRepoImpl{db: db}
}

func (r *notificationSettingRepoImpl) Get(ctx context.Context, accountID, accountUserID string) (*domain.NotificationSetting, *apierror.APIError) {
	ctx, span := notificationSettingRepoTracer.Start(ctx, "repository.notification_setting.get")
	defer span.End()
	out := &domain.NotificationSetting{AccountID: accountID, AccountUserID: accountUserID, Defaults: notifyprefs.Empty()}
	row, err := r.db.GetNotificationSetting(ctx, sqlc.GetNotificationSettingParams{AccountID: accountID, AccountUserID: accountUserID})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		if apiErr.Code == apierror.CodeResourceNotFound {
			return out, nil
		}
		return nil, tracing.Trace(span, apiErr)
	}
	out.Defaults = notifyprefs.Parse(row.Defaults)
	if len(row.AutoSubscribe) > 0 {
		var triggers []constants.AutoSubscribeTrigger
		if err := json.Unmarshal(row.AutoSubscribe, &triggers); err == nil {
			if triggers == nil {
				triggers = []constants.AutoSubscribeTrigger{}
			}
			out.AutoSubscribe = triggers
		}
	}
	if row.PausedUntil.Valid {
		t := row.PausedUntil.Time
		out.PausedUntil = &t
	}
	return out, nil
}

func (r *notificationSettingRepoImpl) Upsert(ctx context.Context, s *domain.NotificationSetting) *apierror.APIError {
	ctx, span := notificationSettingRepoTracer.Start(ctx, "repository.notification_setting.upsert")
	defer span.End()
	rowID, apiErr := id.GenID(id.NotificationPreferenceIDPrefix, nil)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	params := sqlc.UpsertNotificationSettingParams{ID: rowID, AccountID: s.AccountID, AccountUserID: s.AccountUserID}
	if s.AccountUserID == "" {
		params.Defaults = s.Defaults.Marshal()
	}
	if s.AutoSubscribe != nil {
		b, err := json.Marshal(s.AutoSubscribe)
		if err != nil {
			return tracing.Trace(span, apierror.NewInternalError(err, "Failed to encode auto-subscribe settings."))
		}
		params.AutoSubscribe = b
	}
	if s.PausedUntil != nil {
		params.PausedUntil = sql.NullTime{Time: *s.PausedUntil, Valid: true}
	}
	if apiErr := db.MapSQLError(r.db.UpsertNotificationSetting(ctx, params)); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *notificationSettingRepoImpl) RoleDefaults(ctx context.Context, accountUserID string) (notifyprefs.Defaults, *apierror.APIError) {
	ctx, span := notificationSettingRepoTracer.Start(ctx, "repository.notification_setting.role_defaults")
	defer span.End()
	raw, err := r.db.GetAccountUserRoleNotificationDefaults(ctx, accountUserID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		if apiErr.Code == apierror.CodeResourceNotFound {
			return notifyprefs.Empty(), nil
		}
		return notifyprefs.Empty(), tracing.Trace(span, apiErr)
	}
	return notifyprefs.Parse(raw), nil
}
