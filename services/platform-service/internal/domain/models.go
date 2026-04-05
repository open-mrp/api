package domain

import (
	"encoding/json"
	"time"

	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/pagination"
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
	PublicEndpoint       bool
	BodyJSON             *string
	ResponseJSON         *string
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

// RequestLogRead is the enriched read model for request logs, including
// resolved actor details, account name, and idempotency key value.
type RequestLogRead struct {
	ID              string
	Method          string
	Host            string
	Path            string
	NormalizedRoute string
	QueryJSON       *string
	StatusCode      int32
	LatencyUs       int64
	APIVersion      *string
	IdentityType    *string
	ClientIP        *string
	UserAgent       *string
	Referrer        *string
	ErrorCode       *string
	ErrorMessage    *string
	OccurredAt      time.Time
	CreatedAt       time.Time
	AccountID       *string
	AccountName     *string
	Actor           *RequestLogActor
	IdempotencyKey  *string
	BodyJSON        *string
	ResponseJSON    *string
}

type RequestLogActor struct {
	ID            string
	ObjectType    constants.ObjectType
	Name          *string
	Email         *string
	RedactedValue *string
	RoleID        *string
	RoleName      *string
	RoleTypeCode  *string
}

type ListRequestLogsFilter struct {
	Query          *string
	StartDate      *time.Time
	EndDate        *time.Time
	Method         *string
	StatusCode     *int32
	ErrorCode      *string
	AccountID      *string
	ActorID        *string
	ActorType      *string
	ActorName      *string
	ExactMatch     bool
	PublicEndpoint *bool
	Cursor         *string
	Limit          int32
}

type ListRequestLogsResult struct {
	RequestLogs []*RequestLogRead
	PageInfo    pagination.PageInfo
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

// AuditFieldChange represents a before/after transition for a single field.
type AuditFieldChange struct {
	Field    string
	OldValue json.RawMessage
	NewValue json.RawMessage
}

// AuditActor contains the actor details associated with an audit event.
type AuditActor struct {
	ID           string
	ObjectType   constants.ObjectType
	Type         string
	IdentityType string
	Name         *string
	Handle       *string
}

// AuditEvent represents a single immutable audit event record.
// It maps 1:1 with the `audit_event` table for storage.
type AuditEvent struct {
	ID string

	ActorID      string
	ActorType    string
	IdentityType string
	AccountID    string

	Action       constants.AuditAction
	ResourceType constants.ObjectType
	ResourceID   string
	Changes      []AuditFieldChange
	Metadata     json.RawMessage

	ServiceName      string
	RequestID        *string
	IdempotencyKeyID *string
	SourceIP         *string

	OccurredAt time.Time
	CreatedAt  time.Time
}

type AuditEventRead struct {
	AuditEvent
	Actor *AuditActor
}

type ListAuditEventsFilter struct {
	StartDate    *time.Time
	EndDate      *time.Time
	ResourceType *string
	ResourceID   *string
	ActorID      *string
	Action       *string
	AccountID    *string
	Query        *string
	Cursor       *string
	Limit        int32
}

type ListAuditEventsResult struct {
	AuditEvents []*AuditEventRead
	PageInfo    pagination.PageInfo
}
