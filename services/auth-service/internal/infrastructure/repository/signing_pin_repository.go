package repository

import (
	"context"
	"time"

	"github.com/open-mrp/api/services/auth-service/internal/domain"
	"github.com/open-mrp/api/services/auth-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/tracing"
)

var signingPinRepoTracer = tracing.GetTracer("auth-service.signing_pin_repository")

type signingPinRepoImpl struct {
	db *sqlc.Queries
}

func NewSigningPinRepo(db *sqlc.Queries) domain.SigningPinRepo {
	return &signingPinRepoImpl{db: db}
}

func (r *signingPinRepoImpl) FindByUserID(ctx context.Context, userID string) (*domain.SigningPin, *apierror.APIError) {
	ctx, span := signingPinRepoTracer.Start(ctx, "repository.signing_pin.find_by_user_id")
	defer span.End()

	row, err := r.db.GetSigningPinByUserID(ctx, userID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		if apierror.IsNotFound(apiErr) {
			return nil, apiErr
		}
		return nil, tracing.Trace(span, apiErr)
	}

	return &domain.SigningPin{
		UserID:         row.UserID,
		PinHash:        row.PinHash,
		IsTemporary:    row.IsTemporary,
		FailedAttempts: int(row.FailedAttempts),
		LockedUntil:    db.TimeFromNullTime(row.LockedUntil),
		SetAt:          row.SetAt,
	}, nil
}

func (r *signingPinRepoImpl) Upsert(ctx context.Context, userID, pinHash string, isTemporary bool) *apierror.APIError {
	ctx, span := signingPinRepoTracer.Start(ctx, "repository.signing_pin.upsert")
	defer span.End()

	err := r.db.UpsertSigningPin(ctx, sqlc.UpsertSigningPinParams{
		UserID:      userID,
		PinHash:     pinHash,
		IsTemporary: isTemporary,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *signingPinRepoImpl) Delete(ctx context.Context, userID string) (bool, *apierror.APIError) {
	ctx, span := signingPinRepoTracer.Start(ctx, "repository.signing_pin.delete")
	defer span.End()

	rows, err := r.db.DeleteSigningPin(ctx, userID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}
	return rows > 0, nil
}

func (r *signingPinRepoImpl) RecordFailure(ctx context.Context, userID, pinHash string, maxAttempts int, lockUntil time.Time) *apierror.APIError {
	ctx, span := signingPinRepoTracer.Start(ctx, "repository.signing_pin.record_failure")
	defer span.End()

	_, err := r.db.RecordSigningPinFailure(ctx, sqlc.RecordSigningPinFailureParams{
		MaxAttempts: int32(maxAttempts), // #nosec G115 - a small constant attempt limit
		LockUntil:   lockUntil.UTC(),
		UserID:      userID,
		PinHash:     pinHash,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *signingPinRepoImpl) ResetFailures(ctx context.Context, userID string) *apierror.APIError {
	ctx, span := signingPinRepoTracer.Start(ctx, "repository.signing_pin.reset_failures")
	defer span.End()

	err := r.db.ResetSigningPinFailures(ctx, userID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}
