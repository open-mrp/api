package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/timeutil"
)

const SampleJobID = "jb_01k0a5smf9ekb8rqg1"

// Records a piece of work the API accepted and carries out asynchronously. Endpoints
// answering `202 Accepted` point at one with a `Location` header; poll it for the outcome.
type Job struct {
	// Job ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=job"`
	// The kind of work the job carries out.
	Type constants.JobType `json:"type" validate:"required"`
	// How far the job has got. `completed` means the work was processed, not that every
	// row succeeded — read `errors`.
	Status constants.JobStatus `json:"status" validate:"required"`
	// The ID of the account user who requested the work.
	CreatedByID *string `json:"created_by_id"`
	// Name of the account user who requested the work.
	CreatedByName *string `json:"created_by_name"`
	// Username of the account user who requested the work.
	CreatedByUsername *string `json:"created_by_username"`
	// Email of the account user who requested the work.
	CreatedByEmail *string `json:"created_by_email"`
	// One entry per request row that produced a resource. A bulk create records these when
	// it accepts the request, so they stay provisional until `status` is `completed`.
	Results []JobResult `json:"results" expandable:"true"`
	// One entry per failure, so a `completed` job can still carry failed rows. A whole-job
	// failure records a single entry with no `index`.
	Errors []apierror.RowError `json:"errors" expandable:"true"`
	// A one-line reason the last attempt failed.
	ErrorSummary *string `json:"error_summary"`
	// Where a completed export job's file can be downloaded. Null on every other job, and
	// returned only to a caller asking for JSON — otherwise retrieving the job redirects to it.
	Export *JobExport `json:"export"`
	// When the job began executing.
	StartedAt *time.Time `json:"started_at"`
	// When the job finished processing, whether or not every row succeeded.
	CompletedAt *time.Time `json:"completed_at"`
	// When the most recent attempt failed. A retry that succeeds leaves this alongside `completed_at`.
	FailedAt *time.Time `json:"failed_at"`
	// When the job was cancelled.
	CancelledAt *time.Time `json:"cancelled_at"`
	// When the job was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// When the job was last updated.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

// Points a completed export job at the file it produced.
type JobExport struct {
	// Presigned link to the file, valid for five minutes — read the job again for a fresh one.
	// It carries its own authorization, so treat it as a credential: do not log it, store it, or pass it on.
	URL string `json:"url" validate:"required"`
}

// Accounts for one request row that produced a resource. With `errors`, also row-indexed,
// every submitted row lands in exactly one of the two once the job completes.
type JobResult struct {
	// Zero-based row of the request this result names.
	Index int `json:"index"`
	// ID of the resource this row produced.
	ID string `json:"id" validate:"required"`
	// Whether the resource was created or updated.
	Action constants.JobResultAction `json:"action" validate:"required"`
	// Resources created alongside this row's own resource — for a bulk production run
	// create, the ids of its batches. Omitted for operations that produce none.
	SubResourceIDs []string `json:"sub_resource_ids,omitzero"`
}

var SampleJob = &Job{
	ID:     SampleJobID,
	Object: constants.ObjectTypeJob,
	Type:   constants.JobTypeBulkCreate,
	Status: constants.JobStatusCompleted,
	Results: []JobResult{{
		Index:          0,
		ID:             SampleProductionRunID,
		Action:         constants.JobResultActionCreated,
		SubResourceIDs: []string{SampleBatchID},
	}},
	StartedAt:   timeutil.TimestampToTimePtr(sampleCreatedAtTimestamp),
	CompletedAt: timeutil.TimestampToTimePtr(sampleCreatedAtTimestamp),
	CreatedAt:   timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:   timeutil.TimestampToTime(sampleCreatedAtTimestamp),
}

func (*Job) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleJob)
}
