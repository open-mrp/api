package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/augno/api/services/platform-service/internal/domain"
	"github.com/augno/api/services/platform-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/tracing"
)

var idempotencyKeyRepoTracer = tracing.GetTracer("platform-service.idempotency_key_repository")

type idempotencyKeyRepoImpl struct {
	db     *sql.DB
	shared *sqlc.Queries
}

func NewIdempotencyKeyRepo(db *sql.DB, shared *sqlc.Queries) domain.IdempotencyKeyRepo {
	return &idempotencyKeyRepoImpl{db: db, shared: shared}
}

func (r *idempotencyKeyRepoImpl) SetResponse(ctx context.Context, params domain.SetResponseParams) *apierror.APIError {
	ctx, span := idempotencyKeyRepoTracer.Start(ctx, "repository.idempotency_key.set_response")
	defer span.End()

	var err error
	if params.TTLSeconds != nil && *params.TTLSeconds > 0 {
		err = r.shared.SetIdempotencyKeyResponseWithTTL(ctx, sqlc.SetIdempotencyKeyResponseWithTTLParams{
			TypeID:          params.ID,
			ResponseCode:    sql.NullInt32{Int32: int32(params.StatusCode), Valid: true}, // #nosec G115 - HTTP status code
			ResponseBody:    db.NullableRawMessage(params.Body),
			ResponseHeaders: db.NullableRawMessage(params.Headers),
			Column4:         *params.TTLSeconds,
			RecoveryPoint:   params.RecoveryPoint,
		})
	} else {
		err = r.shared.SetIdempotencyKeyResponse(ctx, sqlc.SetIdempotencyKeyResponseParams{
			TypeID:          params.ID,
			ResponseCode:    sql.NullInt32{Int32: int32(params.StatusCode), Valid: true}, // #nosec G115 - HTTP status code
			ResponseBody:    db.NullableRawMessage(params.Body),
			ResponseHeaders: db.NullableRawMessage(params.Headers),
			RecoveryPoint:   params.RecoveryPoint,
		})
	}
	if err != nil {
		return tracing.Trace(span, db.MapSQLError(err))
	}
	return nil
}

const maxDeadlockRetries = 3

func (r *idempotencyKeyRepoImpl) UpsertAndLock(ctx context.Context, key *domain.IdempotencyKey) (*domain.UpsertAndLockResult, *apierror.APIError) {
	ctx, span := idempotencyKeyRepoTracer.Start(ctx, "repository.idempotency_key.upsert_and_lock")
	defer span.End()

	for attempt := 0; attempt <= maxDeadlockRetries; attempt++ {
		result, apiErr, retryable := r.upsertAndLockOnce(ctx, key)
		if apiErr == nil {
			return result, nil
		}
		if !retryable || attempt == maxDeadlockRetries {
			return nil, apiErr
		}
	}

	return nil, apierror.NewInternalError(nil, "Exhausted deadlock retries for idempotency key upsert.")
}

func (r *idempotencyKeyRepoImpl) upsertAndLockOnce(ctx context.Context, key *domain.IdempotencyKey) (*domain.UpsertAndLockResult, *apierror.APIError, bool) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, apierror.NewInternalError(err, "Failed to begin transaction."), false
	}
	defer tx.Rollback()

	txQueries := r.shared.WithTx(tx)

	existing, err := txQueries.GetIdempotencyKeyByScopeHashForUpdate(ctx, key.ScopeHash)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, db.MapSQLError(err), db.IsDeadlock(err)
	}

	if errors.Is(err, sql.ErrNoRows) {
		internalID, createErr := txQueries.CreateIdempotencyKeyWithScope(ctx, sqlc.CreateIdempotencyKeyWithScopeParams{
			TypeID:          key.ID,
			ScopeHash:       key.ScopeHash,
			RequestBodyHash: key.RequestBodyHash,
			ActorID:         db.NullStringPtr(key.ActorID),
			IdentityType:    key.IdentityType,
			TargetAccountID: db.NullStringPtr(key.TargetAccountID),
			RequestMethod:   key.RequestMethod,
			NormalizedRoute: key.NormalizedRoute,
			IdempotencyKey:  key.IdempotencyKey,
			RequestParams:   db.NullableRawMessage(key.RequestParams),
			RecoveryPoint:   key.RecoveryPoint,
			LockOwner:       db.NullStringPtr(key.ActorID),
		})
		if createErr != nil {
			// A concurrent request inserted the same scope_hash first. Treat as retryable so the next attempt finds the existing row.
			return nil, db.MapSQLError(createErr), db.IsDeadlock(createErr) || db.IsDuplicateEntry(createErr)
		}

		if commitErr := tx.Commit(); commitErr != nil {
			return nil, apierror.NewInternalError(commitErr, "Failed to commit transaction."), db.IsDeadlock(commitErr)
		}

		key.InternalID = internalID
		return &domain.UpsertAndLockResult{
			Key:     key,
			Created: true,
			Locked:  true,
		}, nil, false
	}

	existingKey := idempotencyKeyToDomain(&existing)

	if existingKey.RequestBodyHash != key.RequestBodyHash {
		return nil, apierror.NewIdempotencyHashMismatchError(key.IdempotencyKey), false
	}

	if existingKey.HasResponse() {
		if commitErr := tx.Commit(); commitErr != nil {
			return nil, apierror.NewInternalError(commitErr, "Failed to commit transaction."), false
		}
		return &domain.UpsertAndLockResult{
			Key:     existingKey,
			Created: false,
			Locked:  false,
		}, nil, false
	}

	if existingKey.IsLocked() {
		if commitErr := tx.Commit(); commitErr != nil {
			return nil, apierror.NewInternalError(commitErr, "Failed to commit transaction."), false
		}
		return &domain.UpsertAndLockResult{
			Key:     existingKey,
			Created: false,
			Locked:  false,
		}, nil, false
	}

	_, lockErr := txQueries.LockIdempotencyKey(ctx, sqlc.LockIdempotencyKeyParams{
		TypeID:    existingKey.ID,
		LockOwner: db.NullStringPtr(key.ActorID),
	})
	if lockErr != nil {
		return nil, db.MapSQLError(lockErr), db.IsDeadlock(lockErr)
	}

	if commitErr := tx.Commit(); commitErr != nil {
		return nil, apierror.NewInternalError(commitErr, "Failed to commit transaction."), db.IsDeadlock(commitErr)
	}

	return &domain.UpsertAndLockResult{
		Key:     existingKey,
		Created: false,
		Locked:  true,
	}, nil, false
}

func (r *idempotencyKeyRepoImpl) ReleaseLock(ctx context.Context, id string) *apierror.APIError {
	ctx, span := idempotencyKeyRepoTracer.Start(ctx, "repository.idempotency_key.release_lock")
	defer span.End()

	if err := r.shared.ReleaseIdempotencyKeyLock(ctx, id); err != nil {
		return tracing.Trace(span, db.MapSQLError(err))
	}
	return nil
}

func (r *idempotencyKeyRepoImpl) AdvanceRecoveryPoint(ctx context.Context, params domain.AdvanceRecoveryPointParams) *apierror.APIError {
	ctx, span := idempotencyKeyRepoTracer.Start(ctx, "repository.idempotency_key.advance_recovery_point")
	defer span.End()

	if err := r.shared.AdvanceRecoveryPoint(ctx, sqlc.AdvanceRecoveryPointParams{
		RecoveryPoint: params.RecoveryPoint,
		RequestParams: db.NullableRawMessage(params.StepData),
		TypeID:        params.ID,
	}); err != nil {
		return tracing.Trace(span, db.MapSQLError(err))
	}
	return nil
}

func (r *idempotencyKeyRepoImpl) GetRecoveryPoint(ctx context.Context, id string) (*domain.GetRecoveryPointResult, *apierror.APIError) {
	ctx, span := idempotencyKeyRepoTracer.Start(ctx, "repository.idempotency_key.get_recovery_point")
	defer span.End()

	row, err := r.shared.GetRecoveryPoint(ctx, id)
	if err != nil {
		return nil, tracing.Trace(span, db.MapSQLError(err))
	}

	return &domain.GetRecoveryPointResult{
		RecoveryPoint: row.RecoveryPoint,
		StepData:      json.RawMessage(row.RequestParams),
	}, nil
}

func idempotencyKeyToDomain(row *sqlc.IdempotencyKey) *domain.IdempotencyKey {
	key := &domain.IdempotencyKey{
		ID:              row.TypeID,
		InternalID:      row.ID,
		IdempotencyKey:  row.IdempotencyKey,
		ActorID:         db.StringFromNullString(row.ActorID),
		TargetAccountID: db.StringFromNullString(row.TargetAccountID),
		IdentityType:    row.IdentityType,
		RequestMethod:   row.RequestMethod,
		NormalizedRoute: row.NormalizedRoute,
		RequestBodyHash: row.RequestBodyHash,
		ScopeHash:       row.ScopeHash,
		RequestParams:   json.RawMessage(row.RequestParams),
		ResponseBody:    json.RawMessage(row.ResponseBody),
		ResponseHeaders: json.RawMessage(row.ResponseHeaders),
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
		RecoveryPoint:   row.RecoveryPoint,
	}
	if row.ResponseCode.Valid {
		code := int(row.ResponseCode.Int32)
		key.ResponseCode = &code
	}
	if row.LockedAt.Valid {
		key.LockedAt = &row.LockedAt.Time
	}
	if row.LockOwner.Valid {
		key.LockOwner = &row.LockOwner.String
	}
	if row.LockExpiresAt.Valid {
		key.LockExpiresAt = &row.LockExpiresAt.Time
	}
	if row.LastRunAt.Valid {
		key.LastRunAt = row.LastRunAt.Time
	}
	if row.ExpiresAt.Valid {
		key.ExpiresAt = &row.ExpiresAt.Time
	}
	return key
}
