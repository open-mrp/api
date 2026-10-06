package domain

import (
	"context"
	"encoding/json"
	"time"

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
	// ListResourceUserActorIDs returns the distinct user actors that have touched the resource's record tree (events on the resource itself or on children rooted at it). They form the follower set for resource-activity notifications.
	ListResourceUserActorIDs(ctx context.Context, accountID string, resourceType constants.ObjectType, resourceID string) ([]string, *apierror.APIError)
	// GetResourceCreateChanges returns the field changes recorded by the resource's create event (a snapshot of its audited fields at creation), or nil when no create event is on record.
	GetResourceCreateChanges(ctx context.Context, accountID string, resourceType constants.ObjectType, resourceID string) ([]AuditFieldChange, *apierror.APIError)
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

// AccountFollowupRepo persists follow-ups and reads the request activity they are drafted from.
type AccountFollowupRepo interface {
	// Create inserts a scheduled follow-up. A second follow-up for the same account is a no-op, so redelivered schedule commands are harmless.
	Create(ctx context.Context, followup *AccountFollowup) *apierror.APIError
	FindByID(ctx context.Context, id string) (*AccountFollowup, *apierror.APIError)
	// ListDueIDs returns follow-ups ready to draft: scheduled ones whose time has come, and drafts abandoned since staleBefore.
	ListDueIDs(ctx context.Context, now, staleBefore time.Time, limit int32) ([]string, *apierror.APIError)
	// ClaimForDraft moves a due follow-up to drafting, reporting false when another worker already holds it or it is no longer due.
	ClaimForDraft(ctx context.Context, id string, staleBefore time.Time) (bool, *apierror.APIError)
	Skip(ctx context.Context, id string, reason AccountFollowupSkipReason) *apierror.APIError
	Reschedule(ctx context.Context, id string, at time.Time, lastError string) *apierror.APIError
	Fail(ctx context.Context, id string, lastError string) *apierror.APIError
	// SaveDraft stores the draft and moves the follow-up to pending_review, reporting false when it is no longer drafting.
	SaveDraft(ctx context.Context, id string, draft *AccountFollowupDraft) (bool, *apierror.APIError)
	// FindByReviewTokenHash returns the follow-up a review token was issued for.
	FindByReviewTokenHash(ctx context.Context, hash []byte) (*AccountFollowup, *apierror.APIError)
	// Approve records the reviewed text and marks the follow-up sent, reporting false when it is no longer pending review or its review window closed.
	Approve(ctx context.Context, id, subject, body string, now time.Time) (bool, *apierror.APIError)
	// SkipReview marks a pending follow-up skipped by its reviewer, reporting false when it is no longer pending review or its review window closed.
	SkipReview(ctx context.Context, id string, now time.Time) (bool, *apierror.APIError)
	// ListRequests returns up to limit requests made in the account or its sandbox during [from, to), oldest first.
	ListRequests(ctx context.Context, accountID string, sandboxAccountID *string, from, to time.Time, limit int32) ([]AccountRequest, *apierror.APIError)
}
