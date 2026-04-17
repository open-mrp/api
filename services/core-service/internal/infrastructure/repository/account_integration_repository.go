package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/pagination"
	"github.com/augno/api/shared/tracing"
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

	var searchQuery sql.NullString
	if params.Query != nil && *params.Query != "" {
		searchQuery = sql.NullString{String: "%" + *params.Query + "%", Valid: true}
	}

	var cursorDir *pagination.Direction

	if params.Cursor != nil {
		cur, err := pagination.DecodeStringCursor(*params.Cursor)
		if err != nil {
			return nil, apierror.NewValidationError("Invalid pagination cursor.")
		}
		cursorDir = &cur.Direction

		if cur.Direction == pagination.DirectionBackward {
			rows, err := r.queries.ListAccountIntegrationsBackward(ctx, sqlc.ListAccountIntegrationsBackwardParams{
				AccountID:       params.AccountID,
				SearchQuery:     searchQuery,
				CursorCreatedAt: cur.OccurredAt,
				CursorID:        cur.ID,
				Limit:           params.Limit + 1,
			})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}

			integrations := make([]*domain.AccountIntegration, len(rows))
			for i, row := range rows {
				integrations[i] = mapBackwardIntegrationRow(row)
			}

			items, pageInfo := pagination.BuildPageString(integrations, params.Limit, cursorDir, integrationCreatedAt, integrationID)
			return &domain.ListAccountIntegrationsResult{AccountIntegrations: items, PageInfo: pageInfo}, nil
		}

		rows, err := r.queries.ListAccountIntegrationsForward(ctx, sqlc.ListAccountIntegrationsForwardParams{
			AccountID:       params.AccountID,
			SearchQuery:     searchQuery,
			CursorCreatedAt: sql.NullTime{Time: cur.OccurredAt, Valid: true},
			CursorID:        sql.NullString{String: cur.ID, Valid: true},
			Limit:           params.Limit + 1,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}

		integrations := make([]*domain.AccountIntegration, len(rows))
		for i, row := range rows {
			integrations[i] = mapForwardIntegrationRow(row)
		}

		items, pageInfo := pagination.BuildPageString(integrations, params.Limit, cursorDir, integrationCreatedAt, integrationID)
		return &domain.ListAccountIntegrationsResult{AccountIntegrations: items, PageInfo: pageInfo}, nil
	}

	// No cursor — forward query
	rows, err := r.queries.ListAccountIntegrationsForward(ctx, sqlc.ListAccountIntegrationsForwardParams{
		AccountID:   params.AccountID,
		SearchQuery: searchQuery,
		Limit:       params.Limit + 1,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	integrations := make([]*domain.AccountIntegration, len(rows))
	for i, row := range rows {
		integrations[i] = mapForwardIntegrationRow(row)
	}

	items, pageInfo := pagination.BuildPageString(integrations, params.Limit, cursorDir, integrationCreatedAt, integrationID)
	return &domain.ListAccountIntegrationsResult{AccountIntegrations: items, PageInfo: pageInfo}, nil
}

func integrationCreatedAt(item *domain.AccountIntegration) time.Time {
	return item.CreatedAt
}

func integrationID(item *domain.AccountIntegration) string {
	return item.ID
}

func mapForwardIntegrationRow(row sqlc.ListAccountIntegrationsForwardRow) *domain.AccountIntegration {
	return &domain.AccountIntegration{
		ID:              row.ID,
		AccountID:       row.AccountID,
		IntegrationCode: constants.IntegrationCode(row.IntegrationCode),
		Name:            row.Name,
		IsActive:        row.IsActive,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}
}

func mapBackwardIntegrationRow(row sqlc.ListAccountIntegrationsBackwardRow) *domain.AccountIntegration {
	return &domain.AccountIntegration{
		ID:              row.ID,
		AccountID:       row.AccountID,
		IntegrationCode: constants.IntegrationCode(row.IntegrationCode),
		Name:            row.Name,
		IsActive:        row.IsActive,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}
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
		ID:              row.ID,
		AccountID:       row.AccountID,
		IntegrationCode: constants.IntegrationCode(row.IntegrationCode),
		Name:            row.Name,
		IsActive:        row.IsActive,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}, nil
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
		ID:              row.ID,
		AccountID:       row.AccountID,
		IntegrationCode: constants.IntegrationCode(row.IntegrationCode),
		Name:            row.Name,
		IsActive:        row.IsActive,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}, nil
}

func (r *accountIntegrationRepoImpl) Create(ctx context.Context, id string, params domain.CreateAccountIntegrationParams, encryptedCredentials string) (*domain.AccountIntegration, *apierror.APIError) {
	ctx, span := accountIntegrationRepoTracer.Start(ctx, "repository.account_integration.create")
	defer span.End()

	err := r.queries.InsertAccountIntegration(ctx, sqlc.InsertAccountIntegrationParams{
		ID:              id,
		AccountID:       params.AccountID,
		IntegrationCode: string(params.IntegrationCode),
		Name:            params.Name,
		CredentialsV2:   sql.NullString{String: encryptedCredentials, Valid: true},
	})
	if err != nil {
		return nil, tracing.Trace(span, db.MapSQLError(err))
	}

	return r.Get(ctx, params.AccountID, id)
}

func (r *accountIntegrationRepoImpl) UpdateCredentials(ctx context.Context, accountID, id, name, encryptedCredentials string) (*domain.AccountIntegration, *apierror.APIError) {
	ctx, span := accountIntegrationRepoTracer.Start(ctx, "repository.account_integration.update_credentials")
	defer span.End()

	result, err := r.queries.UpdateAccountIntegrationCredentials(ctx, sqlc.UpdateAccountIntegrationCredentialsParams{
		ID:            id,
		AccountID:     accountID,
		Name:          name,
		CredentialsV2: sql.NullString{String: encryptedCredentials, Valid: true},
	})
	if err != nil {
		return nil, tracing.Trace(span, db.MapSQLError(err))
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, tracing.Trace(span, db.MapSQLError(err))
	}
	if rowsAffected == 0 {
		return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Account integration not found."))
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
		return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Account integration not found."))
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
		return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Account integration not found."))
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

	if !row.CredentialsV2.Valid {
		return "", false, tracing.Trace(span, apierror.NewInternalError(nil, "Integration credentials not migrated to envelope format."))
	}

	return row.CredentialsV2.String, row.IsActive, nil
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
