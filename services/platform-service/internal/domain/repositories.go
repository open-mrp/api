package domain

import (
	"context"
	"encoding/json"

	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

type RequestLogRepo interface {
	Create(ctx context.Context, requestLog *RequestLog) *apierror.APIError
	FindByID(ctx context.Context, id, targetAccountID string, includes []string) (*RequestLogRead, *apierror.APIError)
	List(ctx context.Context, targetAccountID string, filter *ListRequestLogsFilter, includes []string) (*ListRequestLogsResult, *apierror.APIError)
}

type AuditEventRepo interface {
	Create(ctx context.Context, event *AuditEvent) *apierror.APIError
	FindByID(ctx context.Context, id, targetAccountID string, includes []string) (*AuditEventRead, *apierror.APIError)
	List(ctx context.Context, targetAccountID string, filter *ListAuditEventsFilter, includes []string) (*ListAuditEventsResult, *apierror.APIError)
	BatchGetResourceCreators(ctx context.Context, callerAccountID, resourceType string, resourceIDs []string) ([]ResourceCreator, *apierror.APIError)
	// GetResourceCreateChanges returns the field changes recorded by the resource's create event (a snapshot of its audited fields at creation), or nil when no create event is on record.
	GetResourceCreateChanges(ctx context.Context, accountID string, resourceType constants.ObjectType, resourceID string) ([]AuditFieldChange, *apierror.APIError)
	// GetResourceCreateEvent returns the resource's create event, or nil when none is on record.
	GetResourceCreateEvent(ctx context.Context, accountID string, resourceType constants.ObjectType, resourceID string) (*ResourceCreateEvent, *apierror.APIError)
	// GetLatestResourceFieldValue returns the field's value as of the resource's newest audit event that recorded it (its create snapshot included), and false when no event recorded it.
	GetLatestResourceFieldValue(ctx context.Context, accountID string, resourceType constants.ObjectType, resourceID, field string) (json.RawMessage, bool, *apierror.APIError)
	// GetAccountUserUserID returns the user behind an account membership, or "" when the membership is not in the account.
	GetAccountUserUserID(ctx context.Context, accountID, accountUserID string) (string, *apierror.APIError)
}

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
	UpsertAndLock(ctx context.Context, key *IdempotencyKey) (*UpsertAndLockResult, *apierror.APIError)
	SetResponse(ctx context.Context, params SetResponseParams) *apierror.APIError
	ReleaseLock(ctx context.Context, id string) *apierror.APIError
	AdvanceRecoveryPoint(ctx context.Context, params AdvanceRecoveryPointParams) *apierror.APIError
	GetRecoveryPoint(ctx context.Context, id string) (*GetRecoveryPointResult, *apierror.APIError)
}
