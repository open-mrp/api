package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

// A one-time HubSpot backfill/reconciliation run for the account.
type HubspotSyncJob struct {
	// HubSpot sync job ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=hubspot_sync_job"`
	// Lifecycle status of the job.
	//
	// - `previewing`: matching customers to HubSpot companies (no writes yet).
	// - `review_pending`: awaiting resolution of ambiguous company matches and confirmation to execute.
	// - `executing`: writing companies, contacts, and deals to HubSpot.
	// - `completed`: finished.
	// - `failed`: stopped on an error (see `last_error`); re-run to resume.
	Status constants.HubspotSyncJobStatus `json:"status" validate:"required"`
	// Whether this run is in dry-run/preview mode.
	DryRun bool `json:"dry_run"`
	// Orders placed on or after this instant are backfilled as Closed-Won deals.
	//
	// When unset, no historical deals are created; companies and contacts still sync.
	GoLiveCutoffAt *time.Time `json:"go_live_cutoff_at"`
	// Dry-run report of what the execute phase will do.
	//
	// Populated once the read-only preview pass finishes.
	Report *HubspotSyncReport `json:"report"`
	// Failure detail when `status` is `failed`.
	LastError *string `json:"last_error"`
	// When the execute phase started.
	StartedAt *time.Time `json:"started_at"`
	// When the job finished.
	CompletedAt *time.Time `json:"completed_at"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

// A tally of what the execute phase would do, produced by the read-only preview pass.
type HubspotSyncReport struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=hubspot_sync_report"`
	// Total customers considered.
	CustomersTotal int `json:"customers_total"`
	// Customers auto-linked to a HubSpot company by a unique domain match.
	CompaniesConfident int `json:"companies_confident"`
	// Customers queued for human company-match review.
	CompaniesAmbiguous int `json:"companies_ambiguous"`
	// Customers with no match — a new company will be created.
	CompaniesToCreate int `json:"companies_to_create"`
	// Customers with an email — contact upsert candidates.
	ContactsWithEmail int `json:"contacts_with_email"`
}

// One customer that needs a human company-match decision before the backfill can write to HubSpot.
type HubspotCompanyReview struct {
	// Review ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=hubspot_company_review"`
	// The sync job this review belongs to.
	Job *HubspotSyncJob `json:"job" validate:"required"`
	// The Augno customer account being matched.
	Customer *Customer `json:"customer" validate:"required"`
	// Candidate HubSpot companies the customer might match.
	Candidates *List[HubspotCompanyCandidate] `json:"candidates"`
	// Resolution status.
	//
	// - `pending`: awaiting a decision.
	// - `resolved`: linked or marked create-new.
	// - `skipped`: excluded from the sync.
	Status constants.HubspotCompanyReviewStatus `json:"status" validate:"required"`
	// How a resolved review was handled.
	//
	// - `link`: the customer was matched to an existing HubSpot company (see `resolved_hubspot_id`).
	// - `create_new`: a new HubSpot company will be created for the customer.
	Resolution *string `json:"resolution"`
	// The HubSpot company id this customer was linked to (when `resolution` is `link`).
	ResolvedHubspotID *string `json:"resolved_hubspot_id"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

// A possible HubSpot company match for a customer.
type HubspotCompanyCandidate struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=hubspot_company_candidate"`
	// HubSpot company id.
	HubspotID string `json:"hubspot_id" validate:"required"`
	// HubSpot company name.
	Name string `json:"name"`
	// HubSpot company domain.
	Domain string `json:"domain"`
}

const SampleHubspotSyncJobID = "igjb_zwfvfjfxl4lj"
const SampleHubspotCompanyReviewID = "igrv_mkhn7eo9qexh"

var SampleHubspotSyncJob = &HubspotSyncJob{
	ID:     SampleHubspotSyncJobID,
	Object: constants.ObjectTypeHubspotSyncJob,
	Status: constants.HubspotSyncJobStatusReviewPending,
	DryRun: true,
	Report: &HubspotSyncReport{
		Object:             constants.ObjectTypeHubspotSyncReport,
		CustomersTotal:     120,
		CompaniesConfident: 80,
		CompaniesAmbiguous: 12,
		CompaniesToCreate:  28,
		ContactsWithEmail:  95,
	},
	CreatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt: timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*HubspotSyncJob) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleHubspotSyncJob)
}

var SampleHubspotCompanyReview = &HubspotCompanyReview{
	ID:       SampleHubspotCompanyReviewID,
	Object:   constants.ObjectTypeHubspotCompanyReview,
	Job:      &HubspotSyncJob{ID: SampleHubspotSyncJobID, Object: constants.ObjectTypeHubspotSyncJob},
	Customer: &Customer{ID: SampleCustomerID, Object: constants.ObjectTypeCustomer, Name: "Acme Manufacturing"},
	Candidates: NewList([]HubspotCompanyCandidate{
		{Object: constants.ObjectTypeHubspotCompanyCandidate, HubspotID: "12345", Name: "Acme Manufacturing", Domain: "acme.com"},
		{Object: constants.ObjectTypeHubspotCompanyCandidate, HubspotID: "67890", Name: "Acme Mfg Inc", Domain: "acme.com"},
	}, PageInfo{}),
	Status:    constants.HubspotCompanyReviewStatusPending,
	CreatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt: timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*HubspotCompanyReview) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleHubspotCompanyReview)
}
