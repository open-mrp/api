package domain

import (
	"context"
	"encoding/json"
	"time"

	apierror "github.com/augno/api/shared/errors"
)

// HubspotSyncSvc is the application service for the HubSpot backfill: starting a job (which dispatches the preview command), reading job status, managing the company-review queue, and triggering execute. The target account and authorization are derived from the request identity.
type HubspotSyncSvc interface {
	StartBackfill(ctx context.Context, params StartHubspotBackfillParams) (*HubspotSyncJob, *apierror.APIError)
	// GetCurrentJob returns the account's most recent backfill job, or a not-found error when none exists. Used by the dashboard to resume an in-progress sync after a refresh.
	GetCurrentJob(ctx context.Context) (*HubspotSyncJob, *apierror.APIError)
	GetJob(ctx context.Context, jobID string) (*HubspotSyncJob, *apierror.APIError)
	ListReviews(ctx context.Context, jobID string, status *string) ([]*HubspotCompanyReview, *apierror.APIError)
	ResolveReview(ctx context.Context, params ResolveHubspotReviewParams) (*HubspotCompanyReview, *apierror.APIError)
	StartExecute(ctx context.Context, jobID string) (*HubspotSyncJob, *apierror.APIError)
}

// StartHubspotBackfillParams starts a backfill. GoLiveCutoffAt bounds which historical orders become deals (nil = no deal backfill).
type StartHubspotBackfillParams struct {
	DryRun         bool
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
	DryRun         bool       `audit:"dry_run"`
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
	DryRun         bool
	GoLiveCutoffAt *time.Time
}

// UpdateHubspotSyncJobParams patches a job. Nil status/cursors/counts/timestamps leave the existing value unchanged; LastError is always written (pass nil to clear it).
type UpdateHubspotSyncJobParams struct {
	ID          string
	AccountID   string
	Status      *string
	Cursors     json.RawMessage
	Counts      json.RawMessage
	LastError   *string
	StartedAt   *time.Time
	CompletedAt *time.Time
}

// HubspotSyncRecord maps one Augno entity to its HubSpot counterpart, making sync idempotent across replays and re-runs.
type HubspotSyncRecord struct {
	ID           string
	AccountID    string
	AugnoType    string
	AugnoID      string
	HubspotType  string
	HubspotID    string
	SyncHash     *string
	LastSyncedAt *time.Time
	LastError    *string
	CreatedAt    time.Time
	UpdatedAt    time.Time
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
