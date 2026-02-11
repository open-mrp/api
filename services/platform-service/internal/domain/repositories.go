package domain

import (
	"context"
	"encoding/json"
	"errors"

	apierror "github.com/augno/api/shared/errors"
)

type RequestLogRepo interface {
	Create(ctx context.Context, requestLog *RequestLog) *apierror.APIError
}

var (
	ErrKeyNotFound  = errors.New("idempotency key not found")
	ErrKeyLocked    = errors.New("idempotency key is locked by another request")
	ErrHashMismatch = errors.New("idempotency key was used with different request parameters")
)

type UpsertAndLockResult struct {
	Key     *IdempotencyKey
	Created bool
	Locked  bool
}

type SetResponseParams struct {
	ID            string
	StatusCode    int
	RecoveryPoint string
	Body          json.RawMessage
	Headers       json.RawMessage
	TTLSeconds    *int32
}

type AdvanceRecoveryPointParams struct {
	ID            string
	RecoveryPoint string
	StepData      json.RawMessage
}

type GetRecoveryPointResult struct {
	RecoveryPoint string
	StepData      json.RawMessage
}

type IdempotencyKeyRepo interface {
	UpsertAndLock(ctx context.Context, key *IdempotencyKey) (*UpsertAndLockResult, error)
	SetResponse(ctx context.Context, params SetResponseParams) error
	ReleaseLock(ctx context.Context, id string) error
	AdvanceRecoveryPoint(ctx context.Context, params AdvanceRecoveryPointParams) error
	GetRecoveryPoint(ctx context.Context, id string) (*GetRecoveryPointResult, error)
}
