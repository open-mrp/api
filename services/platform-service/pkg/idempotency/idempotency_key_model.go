package idempotency

import (
	"encoding/json"
	"time"
)

type IdempotencyKeyNewStatus string

const (
	IdempotencyKeyNewStatusExisting IdempotencyKeyNewStatus = "existing"
	IdempotencyKeyNewStatusNew      IdempotencyKeyNewStatus = "new"
)

type IdempotencyKeyFinishedStatus string

const (
	IdempotencyKeyFinishedStatusIncomplete IdempotencyKeyFinishedStatus = "incomplete"
	IdempotencyKeyFinishedStatusFinished   IdempotencyKeyFinishedStatus = "finished"
)

type IdempotencyKeyLockedStatus string

const (
	IdempotencyKeyLockedStatusUnlocked IdempotencyKeyLockedStatus = "unlocked"
	IdempotencyKeyLockedStatusLocked   IdempotencyKeyLockedStatus = "locked"
)

type CachedResponse struct {
	Code int
	Body json.RawMessage
}

// A representation of an idempotency key that may be used by any other
// service to track the status of an idempotent request.
type IdempotencyKey struct {
	ID             string
	IdempotencyKey string
	NewStatus      IdempotencyKeyNewStatus
	FinishedStatus IdempotencyKeyFinishedStatus
	LockedStatus   IdempotencyKeyLockedStatus
	LastRunAt      time.Time
	LockOwner      *string
	CachedResponse *CachedResponse
}
