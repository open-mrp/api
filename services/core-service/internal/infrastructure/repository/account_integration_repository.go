package repository

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/db"
	"github.com/open-mrp/apikit/pagination"
	"github.com/open-mrp/apikit/tracing"
)

var accountIntegrationRepoTracer = tracing.GetTracer("core-service.account_integration_repository")

type accountIntegrationRepoImpl struct {
	queries *sqlc.Queries
}

func NewAccountIntegrationRepo(queries *sqlc.Queries) domain.AccountIntegrationRepo {
	return &accountIntegrationRepoImpl{queries: queries}
}

func (r *accountIntegrationRepoImpl) List(ctx context.Context, params domain.ListAccountIntegrationsParams) (*domain.ListAccountIntegrationsResult, *apierror.APIError) {
	ctx, span := accountIntegrationRepoTracer.Start(ctx, "repository.account_integration.list")
	defer span.End()

	dir := pagination.DirectionForward
	var cursor *pagination.StringCursor
	var cursorDir *pagination.Direction
	if params.Cursor != nil {
		cur, err := pagination.DecodeStringCursor(*params.Cursor)
		if err != nil {
			return nil, apierror.NewParameterInvalidError("cursor", "Invalid pagination cursor.")
		}
		cursor, cursorDir, dir = &cur, &cur.Direction, cur.Direction
	}

	query, args := buildAccountIntegrationListQuery(params, dir, cursor, params.Limit+1)
	rows, err := r.queries.DB().QueryContext(ctx, query, args...)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	defer func() { _ = rows.Close() }()
	var integrations []*domain.AccountIntegration
	for rows.Next() {
		var ai domain.AccountIntegration
		var code, connection string
		if err := rows.Scan(&ai.ID, &ai.AccountID, &code, &ai.Name, &ai.IsActive, &connection, &ai.CreatedAt, &ai.UpdatedAt); err != nil {
			return nil, tracing.Trace(span, db.MapSQLError(err))
		}
		ai.IntegrationCode = constants.IntegrationCode(code)
		ai.ConnectionStatus = constants.IntegrationConnectionStatus(connection)
		integrations = append(integrations, &ai)
	}
	if apiErr := db.MapSQLError(rows.Err()); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	items, pageInfo := pagination.BuildPageString(integrations, params.Limit, cursorDir, integrationCreatedAt, integrationID)
	return &domain.ListAccountIntegrationsResult{AccountIntegrations: items, PageInfo: pageInfo}, nil
}

// buildAccountIntegrationListQuery emits only the filters the caller supplied. An account has one integration per
// provider, so the scope's unique key reads at most a handful of rows.
func buildAccountIntegrationListQuery(params domain.ListAccountIntegrationsParams, dir pagination.Direction, cursor *pagination.StringCursor, limit int32) (string, []any) {
	where := []string{"account_id = ?"}
	args := []any{params.AccountID}
	if params.Query != nil && *params.Query != "" {
		where, args = append(where, "name LIKE ?"), append(args, "%"+db.EscapeLike(*params.Query)+"%")
	}
	if params.IsActive != nil {
		where, args = append(where, "is_active = ?"), append(args, *params.IsActive)
	}
	order := " ORDER BY created_at DESC, id DESC"
	if cursor != nil {
		if dir == pagination.DirectionBackward {
			where = append(where, "(created_at > ? OR (created_at = ? AND id > ?))")
			order = " ORDER BY created_at ASC, id ASC"
		} else {
			where = append(where, "(created_at < ? OR (created_at = ? AND id < ?))")
		}
		args = append(args, cursor.OccurredAt, cursor.OccurredAt, cursor.ID)
	}
	args = append(args, limit)
	return "SELECT id, account_id, integration_code, name, is_active, connection_status, created_at, updated_at" +
		" FROM account_integration WHERE " + strings.Join(where, " AND ") + order + " LIMIT ?", args
}

func integrationCreatedAt(item *domain.AccountIntegration) time.Time {
	return item.CreatedAt
}

func integrationID(item *domain.AccountIntegration) string {
	return item.ID
}

func (r *accountIntegrationRepoImpl) Get(ctx context.Context, accountID, id string) (*domain.AccountIntegration, *apierror.APIError) {
	ctx, span := accountIntegrationRepoTracer.Start(ctx, "repository.account_integration.get")
	defer span.End()

	row, err := r.queries.GetAccountIntegration(ctx, sqlc.GetAccountIntegrationParams{
		ID:        id,
		AccountID: accountID,
	})
	if err != nil {
		return nil, tracing.Trace(span, db.MapSQLError(err))
	}

	return &domain.AccountIntegration{
		ID:               row.ID,
		AccountID:        row.AccountID,
		IntegrationCode:  constants.IntegrationCode(row.IntegrationCode),
		Name:             row.Name,
		IsActive:         row.IsActive,
		ConnectionStatus: constants.IntegrationConnectionStatus(row.ConnectionStatus),
		CreatedAt:        row.CreatedAt,
		UpdatedAt:        row.UpdatedAt,
	}, nil
}

func (r *accountIntegrationRepoImpl) GetByIDs(ctx context.Context, accountID string, ids []string) ([]*domain.AccountIntegration, *apierror.APIError) {
	ctx, span := accountIntegrationRepoTracer.Start(ctx, "repository.account_integration.get_by_ids")
	defer span.End()

	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := r.queries.GetAccountIntegrationsByIDs(ctx, sqlc.GetAccountIntegrationsByIDsParams{
		Ids:       ids,
		AccountID: accountID,
	})
	if err != nil {
		return nil, tracing.Trace(span, db.MapSQLError(err))
	}
	out := make([]*domain.AccountIntegration, len(rows))
	for i, row := range rows {
		out[i] = &domain.AccountIntegration{
			ID:               row.ID,
			AccountID:        row.AccountID,
			IntegrationCode:  constants.IntegrationCode(row.IntegrationCode),
			Name:             row.Name,
			IsActive:         row.IsActive,
			ConnectionStatus: constants.IntegrationConnectionStatus(row.ConnectionStatus),
			CreatedAt:        row.CreatedAt,
			UpdatedAt:        row.UpdatedAt,
		}
	}
	return out, nil
}

func (r *accountIntegrationRepoImpl) FindByCode(ctx context.Context, accountID string, code constants.IntegrationCode) (*domain.AccountIntegration, *apierror.APIError) {
	ctx, span := accountIntegrationRepoTracer.Start(ctx, "repository.account_integration.find_by_code")
	defer span.End()

	row, err := r.queries.FindAccountIntegrationByCode(ctx, sqlc.FindAccountIntegrationByCodeParams{
		AccountID:       accountID,
		IntegrationCode: string(code),
	})
	if err != nil {
		return nil, tracing.Trace(span, db.MapSQLError(err))
	}

	return &domain.AccountIntegration{
		ID:               row.ID,
		AccountID:        row.AccountID,
		IntegrationCode:  constants.IntegrationCode(row.IntegrationCode),
		Name:             row.Name,
		IsActive:         row.IsActive,
		ConnectionStatus: constants.IntegrationConnectionStatus(row.ConnectionStatus),
		CreatedAt:        row.CreatedAt,
		UpdatedAt:        row.UpdatedAt,
	}, nil
}

func (r *accountIntegrationRepoImpl) Create(ctx context.Context, id string, params domain.CreateAccountIntegrationParams, encryptedCredentials string) (*domain.AccountIntegration, *apierror.APIError) {
	ctx, span := accountIntegrationRepoTracer.Start(ctx, "repository.account_integration.create")
	defer span.End()

	isActive := params.IsActive == nil || *params.IsActive
	err := r.queries.InsertAccountIntegration(ctx, sqlc.InsertAccountIntegrationParams{
		ID:               id,
		AccountID:        params.AccountID,
		IntegrationCode:  string(params.IntegrationCode),
		Name:             params.Name,
		CredentialsV2:    encryptedCredentials,
		IsActive:         isActive,
		ConnectionStatus: string(params.ConnectionStatus),
	})
	if err != nil {
		return nil, tracing.Trace(span, db.MapSQLError(err))
	}

	return r.Get(ctx, params.AccountID, id)
}

func (r *accountIntegrationRepoImpl) UpdateCredentials(ctx context.Context, accountID, id, name, encryptedCredentials string, connection constants.IntegrationConnectionStatus) (*domain.AccountIntegration, *apierror.APIError) {
	ctx, span := accountIntegrationRepoTracer.Start(ctx, "repository.account_integration.update_credentials")
	defer span.End()

	result, err := r.queries.UpdateAccountIntegrationCredentials(ctx, sqlc.UpdateAccountIntegrationCredentialsParams{
		ID:               id,
		AccountID:        accountID,
		Name:             name,
		CredentialsV2:    encryptedCredentials,
		ConnectionStatus: string(connection),
	})
	if err != nil {
		return nil, tracing.Trace(span, db.MapSQLError(err))
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, tracing.Trace(span, db.MapSQLError(err))
	}
	if rowsAffected == 0 {
		return nil, tracing.Trace(span, apierror.NewNotFoundError("Account integration not found."))
	}

	return r.Get(ctx, accountID, id)
}

func (r *accountIntegrationRepoImpl) Update(ctx context.Context, params domain.UpdateAccountIntegrationParams) (*domain.AccountIntegration, *apierror.APIError) {
	ctx, span := accountIntegrationRepoTracer.Start(ctx, "repository.account_integration.update")
	defer span.End()

	var name sql.NullString
	if params.Name != nil {
		name = sql.NullString{String: *params.Name, Valid: true}
	}
	var isActive sql.NullBool
	if params.IsActive != nil {
		isActive = sql.NullBool{Bool: *params.IsActive, Valid: true}
	}

	result, err := r.queries.UpdateAccountIntegration(ctx, sqlc.UpdateAccountIntegrationParams{
		ID:        params.ID,
		AccountID: params.AccountID,
		Name:      name,
		IsActive:  isActive,
	})
	if err != nil {
		return nil, tracing.Trace(span, db.MapSQLError(err))
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, tracing.Trace(span, db.MapSQLError(err))
	}
	if rowsAffected == 0 {
		return nil, tracing.Trace(span, apierror.NewNotFoundError("Account integration not found."))
	}

	return r.Get(ctx, params.AccountID, params.ID)
}

func (r *accountIntegrationRepoImpl) Delete(ctx context.Context, params domain.DeleteAccountIntegrationParams) (*domain.AccountIntegration, *apierror.APIError) {
	ctx, span := accountIntegrationRepoTracer.Start(ctx, "repository.account_integration.delete")
	defer span.End()

	// Fetch the record before deleting (MySQL has no RETURNING)
	integration, apiErr := r.Get(ctx, params.AccountID, params.ID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	result, err := r.queries.DeleteAccountIntegration(ctx, sqlc.DeleteAccountIntegrationParams{
		ID:        params.ID,
		AccountID: params.AccountID,
	})
	if err != nil {
		return nil, tracing.Trace(span, db.MapSQLError(err))
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, tracing.Trace(span, db.MapSQLError(err))
	}
	if rowsAffected == 0 {
		return nil, tracing.Trace(span, apierror.NewNotFoundError("Account integration not found."))
	}

	return integration, nil
}

func (r *accountIntegrationRepoImpl) GetEncryptedCredentials(ctx context.Context, accountID string, code constants.IntegrationCode) (string, bool, *apierror.APIError) {
	ctx, span := accountIntegrationRepoTracer.Start(ctx, "repository.account_integration.get_encrypted_credentials")
	defer span.End()

	row, err := r.queries.GetAccountIntegrationCredentials(ctx, sqlc.GetAccountIntegrationCredentialsParams{
		AccountID:       accountID,
		IntegrationCode: string(code),
	})
	if err != nil {
		return "", false, tracing.Trace(span, db.MapSQLError(err))
	}

	if row.CredentialsV2 == "" {
		return "", false, tracing.Trace(span, apierror.NewInternalError(nil, "Integration credentials not migrated to envelope format."))
	}

	return row.CredentialsV2, row.IsActive, nil
}

func (r *accountIntegrationRepoImpl) HasIntegration(ctx context.Context, accountID string, code constants.IntegrationCode) (bool, *apierror.APIError) {
	ctx, span := accountIntegrationRepoTracer.Start(ctx, "repository.account_integration.has_integration")
	defer span.End()

	count, err := r.queries.CountAccountIntegrationByCode(ctx, sqlc.CountAccountIntegrationByCodeParams{
		AccountID:       accountID,
		IntegrationCode: string(code),
	})
	if err != nil {
		return false, tracing.Trace(span, db.MapSQLError(err))
	}

	return count > 0, nil
}

func (r *accountIntegrationRepoImpl) SetConnectionStatus(ctx context.Context, accountID string, code constants.IntegrationCode, status constants.IntegrationConnectionStatus) *apierror.APIError {
	ctx, span := accountIntegrationRepoTracer.Start(ctx, "repository.account_integration.set_connection_status")
	defer span.End()

	err := r.queries.SetAccountIntegrationConnectionStatus(ctx, sqlc.SetAccountIntegrationConnectionStatusParams{
		ConnectionStatus: string(status),
		AccountID:        accountID,
		IntegrationCode:  string(code),
	})
	return tracing.Trace(span, db.MapSQLError(err))
}
