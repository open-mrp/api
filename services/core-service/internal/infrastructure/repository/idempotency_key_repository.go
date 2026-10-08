package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/blobstore"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/tracing"
)

var idempotencyRepoTracer = tracing.GetTracer("core-service.idempotency_key_repository")

type idempotencyKeyRepoImpl struct {
	queries  *sqlc.Queries
	payloads *Payloads
}

// NewIdempotencyKeyRepo builds the repository. payloads takes cached responses over the inline limit; nil keeps them all in the row.
func NewIdempotencyKeyRepo(queries *sqlc.Queries, payloads *Payloads) domain.IdempotencyKeyRepo {
	return &idempotencyKeyRepoImpl{queries: queries, payloads: payloads}
}

// responseMoveTimeout bounds the post-commit move of a large response into object storage.
const responseMoveTimeout = 30 * time.Second

// responseBodyKey is the object key of a cached response. It derives from the key's own id, so a
// repeated move overwrites the same object.
func responseBodyKey(typeID string) string {
	return "idempotency/" + domain.ServiceName + "/" + typeID + ".json.gz"
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

	key := rowToDomainIdempotencyKey(row)
	if apiErr := r.fillResponseBody(ctx, key, row.ResponseBodyKey); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return key, nil
}

// fillResponseBody reads a response that was moved to object storage. Only a finished key has
// one, so this touches the bucket only on a replay of a large response.
func (r *idempotencyKeyRepoImpl) fillResponseBody(ctx context.Context, key *domain.IdempotencyKey, objectKey sql.NullString) *apierror.APIError {
	if !objectKey.Valid {
		return nil
	}
	body, apiErr := r.payloads.store().Fill(ctx, nil, &objectKey.String)
	if apiErr != nil {
		return apiErr
	}
	key.ResponseBody = body
	return nil
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
		ResponseCode:  sql.NullInt32{Int32: int32(code), Valid: true}, // #nosec G115 - HTTP status code
		ResponseBody:  &body,
		RecoveryPoint: string(recoveryPoint),
		TypeID:        typeID,
	})

	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	if r.payloads.store() != nil && len(body) > blobstore.InlineLimit {
		moveCtx := context.WithoutCancel(ctx)
		db.AfterCommit(ctx, func() {
			go r.moveResponseBody(moveCtx, typeID, body)
		})
	}
	return nil
}

// moveResponseBody moves a committed response body out of its row. The response is written inline
// first, inside the caller's transaction, so caching it never waits on object storage and never
// holds a transaction open; the move runs once that transaction has committed. The object is
// written before the row points at it, and a move that fails part way leaves the body inline, where
// a replay still reads it.
func (r *idempotencyKeyRepoImpl) moveResponseBody(ctx context.Context, typeID string, body json.RawMessage) {
	ctx, cancel := context.WithTimeout(ctx, responseMoveTimeout)
	defer cancel()

	key := responseBodyKey(typeID)
	if apiErr := r.payloads.Store.PutJSON(ctx, key, body); apiErr != nil {
		slog.WarnContext(ctx, "Failed to move idempotent response to object storage; keeping it inline", "error", apiErr, "idempotency_key_id", typeID)
		return
	}
	if err := r.payloads.Pool.MoveIdempotencyResponseToObjectStorage(ctx, sqlc.MoveIdempotencyResponseToObjectStorageParams{
		ResponseBodyKey: sql.NullString{String: key, Valid: true},
		TypeID:          typeID,
	}); err != nil {
		slog.WarnContext(ctx, "Failed to point idempotent response at object storage; keeping it inline", "error", err, "idempotency_key_id", typeID)
	}
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
