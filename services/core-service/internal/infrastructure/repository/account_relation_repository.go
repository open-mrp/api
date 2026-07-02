package repository

import (
	"context"
	"database/sql"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/tracing"
)

var accountRelationRepoTracer = tracing.GetTracer("core-service.account_relation_repository")

type accountRelationRepoImpl struct {
	queries *sqlc.Queries
}

func NewAccountRelationRepo(queries *sqlc.Queries) domain.AccountRelationRepo {
	return &accountRelationRepoImpl{queries: queries}
}

func (r *accountRelationRepoImpl) FindByOwnerAccountAndUserID(ctx context.Context, ownerAccountID, userID string) (*domain.AccountRelation, *apierror.APIError) {
	ctx, span := accountRelationRepoTracer.Start(ctx, "repository.account_relation.find_by_owner_account_and_user_id")
	defer span.End()

	row, err := r.queries.FindAccountRelationByOwnerAccountIDAndUserID(ctx, sqlc.FindAccountRelationByOwnerAccountIDAndUserIDParams{
		OwnerAccountID: ownerAccountID,
		UserID:         userID,
	})

	if apiErr := db.MapSQLError(err); apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			return nil, apiErr
		}
		return nil, tracing.Trace(span, apiErr)
	}

	return &domain.AccountRelation{
		ID:                    row.ID,
		CounterpartyAccountID: row.CounterpartyAccountID,
		RoleCode:              row.AccountRelationRoleCode,
	}, nil
}

func (r *accountRelationRepoImpl) FindByCounterpartyAccountAndUserID(ctx context.Context, counterpartyAccountID, ownerAccountID, userID string) (*domain.AccountRelation, *apierror.APIError) {
	ctx, span := accountRelationRepoTracer.Start(ctx, "repository.account_relation.find_by_counterparty_account_and_user_id")
	defer span.End()

	row, err := r.queries.FindAccountRelationByCounterpartyAccountIDAndUserID(ctx, sqlc.FindAccountRelationByCounterpartyAccountIDAndUserIDParams{
		CounterpartyAccountID: counterpartyAccountID,
		OwnerAccountID:        ownerAccountID,
		UserID:                userID,
	})

	if apiErr := db.MapSQLError(err); apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			return nil, apiErr
		}
		return nil, tracing.Trace(span, apiErr)
	}

	return &domain.AccountRelation{
		ID:                    row.ID,
		OwnerAccountID:        row.OwnerAccountID,
		CounterpartyAccountID: row.CounterpartyAccountID,
		RoleCode:              row.AccountRelationRoleCode,
		IsOwnerSide:           true,
	}, nil
}

func (r *accountRelationRepoImpl) FindByOwnerAccountAndAPIKeyID(ctx context.Context, ownerAccountID string, apiKeyID int64) (*domain.AccountRelation, *apierror.APIError) {
	ctx, span := accountRelationRepoTracer.Start(ctx, "repository.account_relation.find_by_owner_account_and_api_key_id")
	defer span.End()

	row, err := r.queries.FindAccountRelationByOwnerAccountIDAndAPIKeyID(ctx, sqlc.FindAccountRelationByOwnerAccountIDAndAPIKeyIDParams{
		OwnerAccountID: ownerAccountID,
		ID:             apiKeyID,
	})

	if apiErr := db.MapSQLError(err); apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			return nil, apiErr
		}
		return nil, tracing.Trace(span, apiErr)
	}

	return &domain.AccountRelation{
		ID:                    row.ID,
		CounterpartyAccountID: row.CounterpartyAccountID,
		RoleCode:              row.AccountRelationRoleCode,
	}, nil
}

func (r *accountRelationRepoImpl) FindByCounterpartyAccountAndAPIKeyID(ctx context.Context, counterpartyAccountID string, apiKeyID int64) (*domain.AccountRelation, *apierror.APIError) {
	ctx, span := accountRelationRepoTracer.Start(ctx, "repository.account_relation.find_by_counterparty_account_and_api_key_id")
	defer span.End()

	row, err := r.queries.FindAccountRelationByCounterpartyAccountIDAndAPIKeyID(ctx, sqlc.FindAccountRelationByCounterpartyAccountIDAndAPIKeyIDParams{
		CounterpartyAccountID: counterpartyAccountID,
		ID:                    apiKeyID,
	})

	if apiErr := db.MapSQLError(err); apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			return nil, apiErr
		}
		return nil, tracing.Trace(span, apiErr)
	}

	return &domain.AccountRelation{
		ID:                    row.ID,
		OwnerAccountID:        row.OwnerAccountID,
		CounterpartyAccountID: row.CounterpartyAccountID,
		RoleCode:              row.AccountRelationRoleCode,
		IsOwnerSide:           true,
	}, nil
}

func (r *accountRelationRepoImpl) HasRelation(ctx context.Context, ownerAccountID, counterpartyAccountID string) (bool, *apierror.APIError) {
	ctx, span := accountRelationRepoTracer.Start(ctx, "repository.account_relation.has_relation")
	defer span.End()

	hasRelation, err := r.queries.HasRelationByOwnerAndCounterparty(ctx, sqlc.HasRelationByOwnerAndCounterpartyParams{
		OwnerAccountID:        ownerAccountID,
		CounterpartyAccountID: counterpartyAccountID,
	})

	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}

	return hasRelation, nil
}

func (r *accountRelationRepoImpl) CountOtherOwnerRelations(ctx context.Context, counterpartyAccountID, excludeOwnerAccountID string) (int64, *apierror.APIError) {
	ctx, span := accountRelationRepoTracer.Start(ctx, "repository.account_relation.count_other_owner_relations")
	defer span.End()

	count, err := r.queries.CountCounterpartyRelationsExcluding(ctx, sqlc.CountCounterpartyRelationsExcludingParams{
		CounterpartyAccountID: counterpartyAccountID,
		OwnerAccountID:        excludeOwnerAccountID,
	})

	if apiErr := db.MapSQLError(err); apiErr != nil {
		return 0, tracing.Trace(span, apiErr)
	}

	return count, nil
}

func (r *accountRelationRepoImpl) FindRelationByOwnerAndCounterparty(ctx context.Context, ownerAccountID, counterpartyAccountID string) (string, *apierror.APIError) {
	ctx, span := accountRelationRepoTracer.Start(ctx, "repository.account_relation.find_relation_by_owner_and_counterparty")
	defer span.End()

	id, err := r.queries.FindAccountRelationByOwnerAndCounterparty(ctx, sqlc.FindAccountRelationByOwnerAndCounterpartyParams{
		OwnerAccountID:        ownerAccountID,
		CounterpartyAccountID: counterpartyAccountID,
	})

	if apiErr := db.MapSQLError(err); apiErr != nil {
		return "", tracing.Trace(span, apiErr)
	}

	return id, nil
}

func (r *accountRelationRepoImpl) CreateNotificationPreference(ctx context.Context, id, accountRelationID, recipientAccountUserID string, notificationTypeCode string) *apierror.APIError {
	ctx, span := accountRelationRepoTracer.Start(ctx, "repository.account_relation.create_notification_preference")
	defer span.End()

	err := r.queries.InsertAccountRelationNotificationPreference(ctx, sqlc.InsertAccountRelationNotificationPreferenceParams{
		ID:                     id,
		AccountRelationID:      accountRelationID,
		RecipientAccountUserID: recipientAccountUserID,
		NotificationTypeCode:   notificationTypeCode,
	})

	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *accountRelationRepoImpl) ListNotificationPreferences(ctx context.Context, accountRelationID, recipientAccountUserID string) ([]domain.NotificationPreference, *apierror.APIError) {
	ctx, span := accountRelationRepoTracer.Start(ctx, "repository.account_relation.list_notification_preferences")
	defer span.End()

	rows, err := r.queries.ListNotificationPreferences(ctx, sqlc.ListNotificationPreferencesParams{
		AccountRelationID:      accountRelationID,
		RecipientAccountUserID: recipientAccountUserID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	prefs := make([]domain.NotificationPreference, len(rows))
	for i, row := range rows {
		prefs[i] = domain.NotificationPreference{
			ID:                   row.ID,
			NotificationTypeCode: row.NotificationTypeCode,
		}
	}

	return prefs, nil
}

func (r *accountRelationRepoImpl) DeleteNotificationPreference(ctx context.Context, accountRelationID, recipientAccountUserID, notificationTypeCode string) *apierror.APIError {
	ctx, span := accountRelationRepoTracer.Start(ctx, "repository.account_relation.delete_notification_preference")
	defer span.End()

	err := r.queries.DeleteNotificationPreference(ctx, sqlc.DeleteNotificationPreferenceParams{
		AccountRelationID:      accountRelationID,
		RecipientAccountUserID: recipientAccountUserID,
		NotificationTypeCode:   notificationTypeCode,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *accountRelationRepoImpl) FindCustomerByEmail(ctx context.Context, ownerAccountID, email string) (*domain.CustomerByEmail, *apierror.APIError) {
	ctx, span := accountRelationRepoTracer.Start(ctx, "repository.account_relation.find_customer_by_email")
	defer span.End()

	row, err := r.queries.FindCustomerByEmail(ctx, sqlc.FindCustomerByEmailParams{
		OwnerAccountID: ownerAccountID,
		Email:          sql.NullString{String: email, Valid: true},
	})

	if apiErr := db.MapSQLError(err); apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			return nil, nil
		}
		return nil, tracing.Trace(span, apiErr)
	}

	return &domain.CustomerByEmail{
		RelationID:            row.RelationID,
		OwnerAccountID:        row.OwnerAccountID,
		CounterpartyAccountID: row.CounterpartyAccountID,
		RoleCode:              row.AccountRelationRoleCode,
		Alias:                 row.Alias.String,
		Email:                 row.Email.String,
		UserName:              row.UserName.String,
	}, nil
}

func (r *accountRelationRepoImpl) FindContactsByEmail(ctx context.Context, ownerAccountID, email string) ([]domain.ContactMatch, *apierror.APIError) {
	ctx, span := accountRelationRepoTracer.Start(ctx, "repository.account_relation.find_contacts_by_email")
	defer span.End()

	rows, err := r.queries.FindContactsByEmail(ctx, sqlc.FindContactsByEmailParams{
		OwnerAccountID: ownerAccountID,
		Email:          sql.NullString{String: email, Valid: true},
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	matches := make([]domain.ContactMatch, 0, len(rows))
	for _, row := range rows {
		m := domain.ContactMatch{
			AccountUserID: row.AccountUserID,
			UserID:        row.UserID,
			AccountID:     row.AccountID,
			StatusCode:    row.StatusCode,
			CreatedAt:     row.CreatedAt,
			UpdatedAt:     row.UpdatedAt,
			Email:         row.Email.String,
			Relationship:  sqlValueToString(row.Relationship),
		}
		if row.RoleID.Valid {
			roleID := row.RoleID.String
			m.RoleID = &roleID
		}
		if row.DepartmentID.Valid {
			departmentID := row.DepartmentID.String
			m.DepartmentID = &departmentID
		}
		if row.LastUsedAt.Valid {
			lastUsedAt := row.LastUsedAt.Time
			m.LastUsedAt = &lastUsedAt
		}
		matches = append(matches, m)
	}
	return matches, nil
}

// sqlValueToString coerces a dynamically-typed scan target (sqlc types CASE/COALESCE expressions as interface{}) to a string. The MySQL driver yields []byte for character results.
func sqlValueToString(v interface{}) string {
	switch t := v.(type) {
	case string:
		return t
	case []byte:
		return string(t)
	default:
		return ""
	}
}

func (r *accountRelationRepoImpl) FindCustomerAccountsByVendorAndUser(ctx context.Context, vendorAccountID, userID string) ([]domain.CustomerAccountSummary, *apierror.APIError) {
	ctx, span := accountRelationRepoTracer.Start(ctx, "repository.account_relation.find_customer_accounts_by_vendor_and_user")
	defer span.End()

	rows, err := r.queries.FindCustomerAccountsByVendorAndUser(ctx, sqlc.FindCustomerAccountsByVendorAndUserParams{
		UserID:         userID,
		OwnerAccountID: vendorAccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	accounts := make([]domain.CustomerAccountSummary, len(rows))
	for i, row := range rows {
		accounts[i] = domain.CustomerAccountSummary{
			ID:   row.ID,
			Name: row.Name,
		}
	}

	return accounts, nil
}
