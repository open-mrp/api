package domain

import (
	"encoding/json"
	"time"
)

type RequestLog struct {
	ID                   string
	Method               string
	Host                 string
	Path                 string
	NormalizedRoute      string
	QueryJSON            *string
	StatusCode           int32
	LatencyUs            int64
	AccountID            *string
	TargetAccountID      *string
	ClientIP             []byte
	ClientIPString       *string
	UserAgent            *string
	Referrer             *string
	ErrorCode            *string
	ErrorMessage         *string
	CreatedAt            time.Time
	OccurredAt           time.Time
	IdempotencyKeyTypeID *string
	ActorID              *string
	ActorType            *string
	InternalErrorMessage *string
	StackTrace           *string
	IdentityType         *string
	APIVersion           *string
	TraceID              *string
}

type IdempotencyKey struct {
	ID              string
	InternalID      int64
	CreatedAt       time.Time
	UpdatedAt       time.Time
	IdempotencyKey  string
	LastRunAt       time.Time
	LockedAt        *time.Time
	LockOwner       *string
	LockExpiresAt   *time.Time
	RequestMethod   string
	RequestParams   json.RawMessage
	NormalizedRoute string
	ScopeHash       string
	RequestBodyHash string
	ResponseCode    *int
	ResponseBody    json.RawMessage
	ResponseHeaders json.RawMessage
	ActorID         *string
	TargetAccountID *string
	IdentityType    string
	OperationID     *int64
	ExpiresAt       *time.Time
	RecoveryPoint   string
	StepData        json.RawMessage
}

func (k *IdempotencyKey) IsFinished() bool {
	return k.ResponseCode != nil
}

func (k *IdempotencyKey) IsLocked() bool {
	if k.LockExpiresAt == nil {
		return false
	}
	return time.Now().Before(*k.LockExpiresAt)
}

func (k *IdempotencyKey) HasResponse() bool {
	return k.ResponseCode != nil
}
