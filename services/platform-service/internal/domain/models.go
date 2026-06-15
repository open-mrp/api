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
	ID               string
	Method           string
	Host             string
	Path             string
	NormalizedRoute  string
	QueryJSON        *string
	StatusCode       int32
	LatencyUs        int64
	APIVersion       *string
	IdentityType     *string
	ClientIP         *string
	UserAgent        *string
	Referrer         *string
	ErrorCode        *string
	ErrorMessage     *string
	OccurredAt       time.Time
	CreatedAt        time.Time
	AccountID        *string
	AccountName      *string
	AccountCreatedAt *time.Time
	AccountUpdatedAt *time.Time
	Actor            *RequestLogActor
	IdempotencyKey   *string
	BodyJSON         *string
	ResponseJSON     *string
}

type RequestLogActor struct {
	ID            string
	ActorType     constants.ActorType
	Name          *string
	Email         *string
	RedactedValue *string
	RoleID        *string
	RoleName      *string
	RoleType      *string
}

type ListRequestLogsFilter struct {
	Query             *string
	StartDate         *time.Time
	EndDate           *time.Time
	Methods           []string
	StatusCodes       []int32
	StatusCodeClasses []int32
	ErrorCodes        []string
	// ActorAccountIDs filters by request_log.account_id: the account the actor
	// belongs to. TargetAccountIDs filters by request_log.target_account_id: the
	// account the request acted upon. Both narrow within the caller's
	// actor-or-target security scope.
	ActorAccountIDs  []string
	TargetAccountIDs []string
	ActorIDs         []string
	ActorTypes       []string
	NormalizedRoutes []string
	Hosts            []string
	MinLatencyUs     *int64
	PublicEndpoint   *bool
	IdempotencyKey   *string
	Cursor           *string
	Limit            int32
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
	ActorType    constants.ActorType
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
	// TargetAccountID is the account the audited mutation was performed against
	// (the Augno-Account targeted by the originating request). Nullable until the
	// backfill + non-null migration land.
	TargetAccountID *string

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
	// Target account details, resolved from target_account_id for the `account`
	// sub-resource. Nil when the event has no target account (pre-backfill rows).
	AccountName      *string
	AccountCreatedAt *time.Time
	AccountUpdatedAt *time.Time
	IdempotencyKey   *string
}

type ListAuditEventsFilter struct {
	StartDate     *time.Time
	EndDate       *time.Time
	ResourceTypes []string
	ResourceIDs   []string
	ActorIDs      []string
	Actions       []string
	// ActorAccountIDs filters by audit_event.account_id: the account that
	// performed the mutation. TargetAccountIDs filters by
	// audit_event.target_account_id: the account the mutation targeted. Both
	// narrow within the caller's actor-or-target security scope.
	ActorAccountIDs  []string
	TargetAccountIDs []string
	Query            *string
	Cursor           *string
	Limit            int32
}

type ListAuditEventsResult struct {
	AuditEvents []*AuditEventRead
	PageInfo    pagination.PageInfo
}
