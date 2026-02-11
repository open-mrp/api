package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/augno/api/services/platform-service/internal/domain"
	"github.com/augno/api/services/platform-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/db"
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

func (r *idempotencyKeyRepoImpl) SetResponse(ctx context.Context, params domain.SetResponseParams) error {
	ctx, span := idempotencyKeyRepoTracer.Start(ctx, "repository.idempotency_key.set_response")
	defer span.End()

	if params.TTLSeconds != nil && *params.TTLSeconds > 0 {
		return r.shared.SetIdempotencyKeyResponseWithTTL(ctx, sqlc.SetIdempotencyKeyResponseWithTTLParams{
			TypeID:          params.ID,
			ResponseCode:    sql.NullInt32{Int32: int32(params.StatusCode), Valid: true}, // #nosec G115 - HTTP status code
			ResponseBody:    db.NullableRawMessage(params.Body),
			ResponseHeaders: db.NullableRawMessage(params.Headers),
			DATEADD:         *params.TTLSeconds,
			RecoveryPoint:   params.RecoveryPoint,
		})
	}

	return r.shared.SetIdempotencyKeyResponse(ctx, sqlc.SetIdempotencyKeyResponseParams{
		TypeID:          params.ID,
		ResponseCode:    sql.NullInt32{Int32: int32(params.StatusCode), Valid: true}, // #nosec G115 - HTTP status code
		ResponseBody:    db.NullableRawMessage(params.Body),
		ResponseHeaders: db.NullableRawMessage(params.Headers),
		RecoveryPoint:   params.RecoveryPoint,
	})
}

func (r *idempotencyKeyRepoImpl) UpsertAndLock(ctx context.Context, key *domain.IdempotencyKey) (*domain.UpsertAndLockResult, error) {
	ctx, span := idempotencyKeyRepoTracer.Start(ctx, "repository.idempotency_key.upsert_and_lock")
	defer span.End()

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	txQueries := r.shared.WithTx(tx)

	existing, err := txQueries.GetIdempotencyKeyByScopeHashForUpdate(ctx, key.ScopeHash)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
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
		})
		if createErr != nil {
			return nil, createErr
		}

		if commitErr := tx.Commit(); commitErr != nil {
			return nil, commitErr
		}

		key.InternalID = internalID
		return &domain.UpsertAndLockResult{
			Key:     key,
			Created: true,
			Locked:  true,
		}, nil
	}

	existingKey := idempotencyKeyToDomain(&existing)

	if existingKey.RequestBodyHash != key.RequestBodyHash {
		return nil, domain.ErrHashMismatch
	}

	if existingKey.HasResponse() {
		if commitErr := tx.Commit(); commitErr != nil {
			return nil, commitErr
		}
		return &domain.UpsertAndLockResult{
			Key:     existingKey,
			Created: false,
			Locked:  false,
		}, nil
	}

	if existingKey.IsLocked() {
		if commitErr := tx.Commit(); commitErr != nil {
			return nil, commitErr
		}
		return &domain.UpsertAndLockResult{
			Key:     existingKey,
			Created: false,
			Locked:  false,
		}, nil
	}

	_, lockErr := txQueries.LockIdempotencyKey(ctx, sqlc.LockIdempotencyKeyParams{
		TypeID:    existingKey.ID,
		LockOwner: db.NullStringPtr(key.ActorID),
	})
	if lockErr != nil {
		return nil, lockErr
	}

	if commitErr := tx.Commit(); commitErr != nil {
		return nil, commitErr
	}

	return &domain.UpsertAndLockResult{
		Key:     existingKey,
		Created: false,
		Locked:  true,
	}, nil
}

func (r *idempotencyKeyRepoImpl) ReleaseLock(ctx context.Context, id string) error {
	ctx, span := idempotencyKeyRepoTracer.Start(ctx, "repository.idempotency_key.release_lock")
	defer span.End()

	return r.shared.ReleaseIdempotencyKeyLock(ctx, id)
}

func (r *idempotencyKeyRepoImpl) AdvanceRecoveryPoint(ctx context.Context, params domain.AdvanceRecoveryPointParams) error {
	ctx, span := idempotencyKeyRepoTracer.Start(ctx, "repository.idempotency_key.advance_recovery_point")
	defer span.End()

	return r.shared.AdvanceRecoveryPoint(ctx, sqlc.AdvanceRecoveryPointParams{
		RecoveryPoint: params.RecoveryPoint,
		RequestParams: db.NullableRawMessage(params.StepData),
		TypeID:        params.ID,
	})
}

func (r *idempotencyKeyRepoImpl) GetRecoveryPoint(ctx context.Context, id string) (*domain.GetRecoveryPointResult, error) {
	ctx, span := idempotencyKeyRepoTracer.Start(ctx, "repository.idempotency_key.get_recovery_point")
	defer span.End()

	row, err := r.shared.GetRecoveryPoint(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrKeyNotFound
		}
		return nil, err
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
