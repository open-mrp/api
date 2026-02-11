package repository

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/augno/api/services/notification-service/internal/domain"
	"github.com/augno/api/services/notification-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/tracing"
)

var idempotencyRepoTracer = tracing.GetTracer("notification-service.idempotency_key_repository")

type idempotencyKeyRepoImpl struct {
	queries *sqlc.Queries
}

func NewIdempotencyKeyRepo(queries *sqlc.Queries) domain.IdempotencyKeyRepo {
	return &idempotencyKeyRepoImpl{queries: queries}
}

func (r *idempotencyKeyRepoImpl) GetByScopeHash(ctx context.Context, scopeHash string) (*domain.IdempotencyKey, *apierror.APIError) {
	ctx, span := idempotencyRepoTracer.Start(ctx, "repository.idempotency_key.get_by_scope_hash")
	defer span.End()

	row, err := r.queries.GetIdempotencyKeyByScopeHash(ctx, sqlc.GetIdempotencyKeyByScopeHashParams{
		ServiceName: domain.ServiceName,
		ScopeHash:   scopeHash,
	})

	if apiErr := db.MapSQLError(err); apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			return nil, apiErr
		}
		return nil, tracing.Trace(span, apiErr)
	}

	return rowToDomainIdempotencyKey(row), nil
}

func (r *idempotencyKeyRepoImpl) Create(ctx context.Context, key *domain.IdempotencyKey) (*domain.IdempotencyKey, *apierror.APIError) {
	ctx, span := idempotencyRepoTracer.Start(ctx, "repository.idempotency_key.create")
	defer span.End()

	internalID, err := r.queries.CreateIdempotencyKey(ctx, sqlc.CreateIdempotencyKeyParams{
		TypeID:         key.TypeID,
		ServiceName:    key.ServiceName,
		Handler:        key.Handler,
		IdempotencyKey: key.IdempotencyKey,
		ActorID:        db.NullStringPtr(key.ActorID),
		IdentityType:   key.IdentityType,
		ScopeHash:      key.ScopeHash,
		RecoveryPoint:  key.RecoveryPoint,
	})

	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return &domain.IdempotencyKey{
		ID:             internalID,
		TypeID:         key.TypeID,
		ServiceName:    key.ServiceName,
		Handler:        key.Handler,
		IdempotencyKey: key.IdempotencyKey,
		ActorID:        key.ActorID,
		IdentityType:   key.IdentityType,
		ScopeHash:      key.ScopeHash,
		RecoveryPoint:  key.RecoveryPoint,
	}, nil
}

func (r *idempotencyKeyRepoImpl) AdvanceRecoveryPoint(ctx context.Context, typeID string, recoveryPoint domain.RecoveryPoint) *apierror.APIError {
	ctx, span := idempotencyRepoTracer.Start(ctx, "repository.idempotency_key.advance_recovery_point")
	defer span.End()

	err := r.queries.AdvanceIdempotencyRecoveryPoint(ctx, sqlc.AdvanceIdempotencyRecoveryPointParams{
		RecoveryPoint: string(recoveryPoint),
		TypeID:        typeID,
	})

	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *idempotencyKeyRepoImpl) GetRecoveryPoint(ctx context.Context, typeID string) (string, *apierror.APIError) {
	ctx, span := idempotencyRepoTracer.Start(ctx, "repository.idempotency_key.get_recovery_point")
	defer span.End()

	recoveryPoint, err := r.queries.GetIdempotencyRecoveryPoint(ctx, typeID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return "", tracing.Trace(span, apiErr)
	}

	return recoveryPoint, nil
}

func (r *idempotencyKeyRepoImpl) SetResponse(ctx context.Context, typeID string, code int, body json.RawMessage, recoveryPoint domain.RecoveryPoint) *apierror.APIError {
	ctx, span := idempotencyRepoTracer.Start(ctx, "repository.idempotency_key.set_response")
	defer span.End()

	err := r.queries.SetIdempotencyResponse(ctx, sqlc.SetIdempotencyResponseParams{
		ResponseCode:  sql.NullInt32{Int32: int32(code), Valid: true}, // #nosec G115 - HTTP status code
		ResponseBody:  &body,
		RecoveryPoint: string(recoveryPoint),
		TypeID:        typeID,
	})

	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func rowToDomainIdempotencyKey(row sqlc.ServiceIdempotencyKey) *domain.IdempotencyKey {
	var responseCode *int
	if row.ResponseCode.Valid {
		code := int(row.ResponseCode.Int32)
		responseCode = &code
	}
	var responseBody json.RawMessage
	if row.ResponseBody != nil {
		responseBody = *row.ResponseBody
	}

	return &domain.IdempotencyKey{
		ID:             row.ID,
		TypeID:         row.TypeID,
		ServiceName:    row.ServiceName,
		Handler:        row.Handler,
		IdempotencyKey: row.IdempotencyKey,
		ActorID:        db.StringFromNullString(row.ActorID),
		IdentityType:   row.IdentityType,
		ScopeHash:      row.ScopeHash,
		ResponseCode:   responseCode,
		ResponseBody:   responseBody,
		RecoveryPoint:  row.RecoveryPoint,
	}
}
