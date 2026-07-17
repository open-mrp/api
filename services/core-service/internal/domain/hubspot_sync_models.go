package domain

import (
	"context"
	"encoding/json"
	"time"

	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/pagination"
)

// HubspotSyncSvc is the application service for the HubSpot backfill: starting a job (which dispatches the preview command), reading job status, managing the company-review queue, and triggering execute. The target account and authorization are derived from the request identity.
type HubspotSyncSvc interface {
	StartBackfill(ctx context.Context, params StartHubspotBackfillParams) (*HubspotSyncJob, *apierror.APIError)
	// GetCurrentJob returns the account's most recent backfill job, or a not-found error when none exists. Used by the dashboard to resume an in-progress sync after a refresh.
	GetCurrentJob(ctx context.Context) (*HubspotSyncJob, *apierror.APIError)
	GetJob(ctx context.Context, jobID string) (*HubspotSyncJob, *apierror.APIError)
	ListReviews(ctx context.Context, jobID string, status *string) ([]*HubspotCompanyReview, *apierror.APIError)
	// ListRecords returns what the sync has actually written to HubSpot for the caller's account — the mapping the engine keeps, which is otherwise invisible.
	ListRecords(ctx context.Context, params ListHubspotSyncRecordsParams) (*ListHubspotSyncRecordsResult, *apierror.APIError)
	ResolveReview(ctx context.Context, params ResolveHubspotReviewParams) (*HubspotCompanyReview, *apierror.APIError)
	StartExecute(ctx context.Context, jobID string) (*HubspotSyncJob, *apierror.APIError)
	// CancelJob force-fails an in-flight job, releasing the account to start a new backfill after a worker died without recording an outcome.
	CancelJob(ctx context.Context, jobID string) (*HubspotSyncJob, *apierror.APIError)
}

// StartHubspotBackfillParams starts a backfill. GoLiveCutoffAt bounds which historical orders become deals (nil = no deal backfill).
type StartHubspotBackfillParams struct {
	GoLiveCutoffAt *time.Time
}

// ResolveHubspotReviewParams resolves one ambiguous company review. Action is "link" (requires ResolvedHubspotID), "create_new", or "skip".
type ResolveHubspotReviewParams struct {
	ReviewID          string
	Action            string
	ResolvedHubspotID *string
}

// HubspotSyncJob is one backfill/reconciliation run for an account. See the hubspot_sync_job table and the hubspotsync package for the state machine.
type HubspotSyncJob struct {
	ID             string
	AccountID      string     `audit:"account_id"`
	Status         string     `audit:"status"`
	GoLiveCutoffAt *time.Time `audit:"go_live_cutoff_at"`
	Cursors        json.RawMessage
	Counts         json.RawMessage
	LastError      *string    `audit:"last_error"`
	StartedAt      *time.Time `audit:"started_at"`
	CompletedAt    *time.Time `audit:"completed_at"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type CreateHubspotSyncJobParams struct {
	AccountID      string
	Status         string
	GoLiveCutoffAt *time.Time
}

// UpdateHubspotSyncJobParams patches a job. Nil fields leave the existing value unchanged, so a partial write (a cursor checkpoint, say) cannot erase an unrelated column.
type UpdateHubspotSyncJobParams struct {
	ID        string
	AccountID string
	Status    *string
	Cursors   json.RawMessage
	Counts    json.RawMessage
	// LastError is three-way: nil preserves the stored error, a non-empty string replaces it, and an empty string clears it.
	LastError   *string
	StartedAt   *time.Time
	CompletedAt *time.Time
}

// HubspotSyncRecord maps one Augno entity to its HubSpot counterpart, making sync idempotent across replays and re-runs.
type HubspotSyncRecord struct {
	ID        string
	AccountID string
	AugnoType string
	AugnoID   string
	// AugnoName is the display name of the mapped Augno entity, resolved by the list query. Empty when the entity no longer exists or was not joined.
	AugnoName    string
	HubspotType  string
	HubspotID    string
	SyncHash     *string
	LastSyncedAt *time.Time
	LastError    *string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// ListHubspotSyncRecordsParams pages the account's Augno->HubSpot mappings. AugnoType is required: it keeps the keyset on the (account_id, augno_type, augno_id) index.
type ListHubspotSyncRecordsParams struct {
	AccountID string
	AugnoType string
	Cursor    *string
	Limit     int32
}

// ListHubspotSyncRecordsResult is one page of mappings.
type ListHubspotSyncRecordsResult struct {
	Items    []*HubspotSyncRecord
	PageInfo pagination.PageInfo
}

type UpsertHubspotSyncRecordParams struct {
	AccountID   string
	AugnoType   string
	AugnoID     string
	HubspotType string
	HubspotID   string
	SyncHash    *string
}

// HubspotCompanyReview is one customer that needs human resolution before the backfill can create/link its HubSpot company.
type HubspotCompanyReview struct {
	ID                string
	JobID             string `audit:"job_id"`
	AccountID         string `audit:"account_id"`
	AugnoCustomerID   string `audit:"augno_customer_id"`
	CustomerName      string `audit:"customer_name"`
	CandidateMatches  json.RawMessage
	Status            string  `audit:"status"`
	Resolution        *string `audit:"resolution"`
	ResolvedHubspotID *string `audit:"resolved_hubspot_id"`
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type CreateHubspotCompanyReviewParams struct {
	JobID            string
	AccountID        string
	AugnoCustomerID  string
	CustomerName     string
	CandidateMatches json.RawMessage
	Status           string
}

type ResolveHubspotCompanyReviewParams struct {
	ID                string
	AccountID         string
	Status            string
	Resolution        *string
	ResolvedHubspotID *string
}
