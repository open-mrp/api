package domain

import "time"

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
	IdempotencyKeyID     *string
	ActorID              *string
	ActorType            *string
	InternalErrorMessage *string
	StackTrace           *string
	IdentityType         *string
}
