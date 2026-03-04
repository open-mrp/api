package repository

import (
	"context"

	"github.com/augno/api/services/auth-service/internal/apikey"
	"github.com/augno/api/services/auth-service/internal/domain"
	"github.com/augno/api/services/auth-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/tracing"
)

var docAPIKeyRepoTracer = tracing.GetTracer("auth-service.doc_api_key_repository")

type docAPIKeyRepoImpl struct {
	queries *sqlc.Queries
}

func NewDocAPIKeyRepo(queries *sqlc.Queries) domain.DocAPIKeyRepo {
	return &docAPIKeyRepoImpl{queries: queries}
}

func (r *docAPIKeyRepoImpl) FindBySandboxAccountID(ctx context.Context, sandboxAccountID string) (*apikey.DocAPIKey, *apierror.APIError) {
	ctx, span := docAPIKeyRepoTracer.Start(ctx, "repository.doc_api_key.find_by_sandbox_account_id")
	defer span.End()

	row, err := r.queries.FindDocAPIKeyBySandboxAccountID(ctx, sandboxAccountID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return &apikey.DocAPIKey{
		ID:              row.ID,
		TypeID:          row.TypeID,
		APIKeyID:        row.ApiKeyID,
		EncryptedSecret: row.EncryptedSecret,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
		APIKeyExpiresAt: db.TimeFromNullTime(row.AkExpiresAt),
		APIKeyRevokedAt: db.TimeFromNullTime(row.AkRevokedAt),
	}, nil
}

func (r *docAPIKeyRepoImpl) FindByAPIKeyID(ctx context.Context, apiKeyID string) (*apikey.DocAPIKey, *apierror.APIError) {
	ctx, span := docAPIKeyRepoTracer.Start(ctx, "repository.doc_api_key.find_by_api_key_id")
	defer span.End()

	row, err := r.queries.FindDocAPIKeyByAPIKeyID(ctx, apiKeyID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return &apikey.DocAPIKey{
		ID:              row.ID,
		TypeID:          row.TypeID,
		APIKeyID:        row.ApiKeyID,
		EncryptedSecret: row.EncryptedSecret,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}, nil
}

func (r *docAPIKeyRepoImpl) Create(ctx context.Context, docAPIKey *apikey.DocAPIKey) (int64, *apierror.APIError) {
	ctx, span := docAPIKeyRepoTracer.Start(ctx, "repository.doc_api_key.create")
	defer span.End()

	result, err := r.queries.CreateDocAPIKey(ctx, sqlc.CreateDocAPIKeyParams{
		TypeID:          docAPIKey.TypeID,
		ApiKeyID:        docAPIKey.APIKeyID,
		EncryptedSecret: docAPIKey.EncryptedSecret,
	})

	if apiErr := db.MapSQLError(err); apiErr != nil {
		return 0, tracing.Trace(span, apiErr)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, tracing.Trace(span, apierror.NewInternalError(err, "failed to get last insert id"))
	}

	return id, nil
}

func (r *docAPIKeyRepoImpl) Update(ctx context.Context, docAPIKey *apikey.DocAPIKey) *apierror.APIError {
	ctx, span := docAPIKeyRepoTracer.Start(ctx, "repository.doc_api_key.update")
	defer span.End()

	err := r.queries.UpdateDocAPIKey(ctx, sqlc.UpdateDocAPIKeyParams{
		ApiKeyID:        docAPIKey.APIKeyID,
		EncryptedSecret: docAPIKey.EncryptedSecret,
		ID:              docAPIKey.ID,
	})

	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *docAPIKeyRepoImpl) Delete(ctx context.Context, id int64) *apierror.APIError {
	ctx, span := docAPIKeyRepoTracer.Start(ctx, "repository.doc_api_key.delete")
	defer span.End()

	err := r.queries.DeleteDocAPIKeyByID(ctx, id)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

