package repository

import (
	"context"
	"encoding/json"

	"github.com/open-mrp/api/services/agent-service/internal/domain"
	agentdb "github.com/open-mrp/api/services/agent-service/internal/infrastructure/db"
	"github.com/open-mrp/api/services/agent-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/safeconv"
	"github.com/open-mrp/api/shared/tracing"
)

var idempotencyRepoTracer = tracing.GetTracer("agent-service.idempotency_key_repository")

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
		ActorID:        agentdb.PgTextPtr(key.ActorID),
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

func (r *idempotencyKeyRepoImpl) GetRecoveryPoint(ctx context.Context, typeID string) (domain.RecoveryPoint, *apierror.APIError) {
	ctx, span := idempotencyRepoTracer.Start(ctx, "repository.idempotency_key.get_recovery_point")
	defer span.End()

	rawRecoveryPoint, err := r.queries.GetIdempotencyRecoveryPoint(ctx, typeID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return "", tracing.Trace(span, apiErr)
	}

	recoveryPoint := domain.RecoveryPoint(rawRecoveryPoint)
	if ok := domain.RecoveryPoint.IsValid(recoveryPoint); ok {
		return recoveryPoint, nil
	}
	return "", tracing.Trace(span, apierror.NewInvariantViolationError("recovery point pulled from database is not valid"))
}

func (r *idempotencyKeyRepoImpl) SetResponse(ctx context.Context, typeID string, code int, body json.RawMessage, recoveryPoint domain.RecoveryPoint) *apierror.APIError {
	ctx, span := idempotencyRepoTracer.Start(ctx, "repository.idempotency_key.set_response")
	defer span.End()

	err := r.queries.SetIdempotencyResponse(ctx, sqlc.SetIdempotencyResponseParams{
		ResponseCode:  agentdb.PgInt4(safeconv.IntToInt32(code)),
		ResponseBody:  body,
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

	return &domain.IdempotencyKey{
		ID:             row.ID,
		TypeID:         row.TypeID,
		ServiceName:    row.ServiceName,
		Handler:        row.Handler,
		IdempotencyKey: row.IdempotencyKey,
		ActorID:        agentdb.StringFromPgText(row.ActorID),
		IdentityType:   row.IdentityType,
		ScopeHash:      row.ScopeHash,
		ResponseCode:   responseCode,
		ResponseBody:   row.ResponseBody,
		RecoveryPoint:  row.RecoveryPoint,
	}
}
