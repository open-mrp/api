package mediator

import (
	"context"
	"fmt"
	"time"

	"github.com/open-mrp/api/services/auth-service/internal/domain"
	"github.com/open-mrp/api/shared/crypto"
	apierror "github.com/open-mrp/api/shared/errors"
	tracing "github.com/open-mrp/api/shared/tracing"
)

const (
	signingPinMaxFailedAttempts = 5
	signingPinLockout           = 15 * time.Minute
)

var signingPinMedTracer = tracing.GetTracer("auth-service.signing_pin_mediator")

type signingPinMedImpl struct {
	repos domain.RepoFactory
	now   func() time.Time
}

type SigningPinMedConfig struct {
	// Repos (required) is the repository factory for signing PIN persistence.
	Repos domain.RepoFactory

	// Now (optional; default: time.Now) is the clock lockouts are measured against.
	Now func() time.Time
}

func (c *SigningPinMedConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("signing pin mediator: repos is required")
	}
	return nil
}

func NewSigningPinMed(config *SigningPinMedConfig) domain.SigningPinMed {
	if err := config.validate(); err != nil {
		panic(err)
	}

	now := config.Now
	if now == nil {
		now = time.Now
	}

	return &signingPinMedImpl{
		repos: config.Repos,
		now:   now,
	}
}

// Check compares pin with the user's signing PIN without recording the outcome.
//
// 1. Load the user's PIN; fail with signing_pin_not_set when they have none.
// 2. Fail with signing_pin_locked while a lockout is in force.
// 3. Compare pin with the stored bcrypt hash.
func (m *signingPinMedImpl) Check(ctx context.Context, userID, pin string) (*domain.SigningPinCheck, *apierror.APIError) {
	ctx, span := signingPinMedTracer.Start(ctx, "mediator.signing_pin.check")
	defer span.End()

	stored, apiErr := m.repos.NewSigningPinRepo().FindByUserID(ctx, userID)
	if apiErr != nil {
		if apierror.IsNotFound(apiErr) {
			return nil, apierror.NewSigningPINNotSetError(ErrSigningPinNotSet)
		}
		return nil, tracing.Trace(span, apiErr)
	}

	if stored.IsLocked(m.now()) {
		return nil, apierror.NewSigningPINLockedError(ErrSigningPinLocked)
	}

	match, err := crypto.CompareBcryptHash(stored.PinHash, pin)
	if err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to compare signing PIN."))
	}

	return &domain.SigningPinCheck{Pin: stored, Matched: match}, nil
}

// RecordFailure counts one wrong PIN against the PIN that was checked and returns the resulting lockout state.
//
// 1. Count the attempt in a single statement; the fifth wrong PIN in a row locks signing for 15 minutes and restarts the count.
// 2. Reload the PIN to report whether this attempt locked it.
func (m *signingPinMedImpl) RecordFailure(ctx context.Context, pin *domain.SigningPin) (*domain.SigningPinFailure, *apierror.APIError) {
	ctx, span := signingPinMedTracer.Start(ctx, "mediator.signing_pin.record_failure")
	defer span.End()

	repo := m.repos.NewSigningPinRepo()
	now := m.now()
	if apiErr := repo.RecordFailure(ctx, pin.UserID, pin.PinHash, signingPinMaxFailedAttempts, now.Add(signingPinLockout)); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	stored, apiErr := repo.FindByUserID(ctx, pin.UserID)
	if apiErr != nil {
		if apierror.IsNotFound(apiErr) {
			return &domain.SigningPinFailure{}, nil
		}
		return nil, tracing.Trace(span, apiErr)
	}

	failure := &domain.SigningPinFailure{FailedAttempts: stored.FailedAttempts}
	if stored.IsLocked(now) {
		failure.LockedUntil = stored.LockedUntil
	}
	return failure, nil
}

// ResetFailures clears the failed attempts a correct PIN follows.
//
// 1. Skip the write when nothing is counted.
// 2. Otherwise clear the count and any expired lockout.
func (m *signingPinMedImpl) ResetFailures(ctx context.Context, pin *domain.SigningPin) *apierror.APIError {
	ctx, span := signingPinMedTracer.Start(ctx, "mediator.signing_pin.reset_failures")
	defer span.End()

	if pin.FailedAttempts == 0 && pin.LockedUntil == nil {
		return nil
	}
	if apiErr := m.repos.NewSigningPinRepo().ResetFailures(ctx, pin.UserID); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

// Save hashes pin and stores it as the user's signing PIN.
//
// 1. Hash pin with bcrypt.
// 2. Replace the user's PIN, clearing failed attempts and any lockout.
func (m *signingPinMedImpl) Save(ctx context.Context, userID, pin string, isTemporary bool) *apierror.APIError {
	ctx, span := signingPinMedTracer.Start(ctx, "mediator.signing_pin.save")
	defer span.End()

	hash, err := crypto.HashBcrypt(pin)
	if err != nil {
		return tracing.Trace(span, apierror.NewInternalError(err, "Failed to hash signing PIN."))
	}

	if apiErr := m.repos.NewSigningPinRepo().Upsert(ctx, userID, hash, isTemporary); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}
