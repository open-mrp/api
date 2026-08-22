package apiresource

import (
	"time"

	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/timeutil"
)

const SampleJobID = "jb_grz7cdpnz8jr"

// Records a piece of work the API accepted and carries out asynchronously. Endpoints
// answering `202 Accepted` point at one with a `Location` header; poll it for the outcome.
type Job struct {
	// Job ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=job"`
	// The kind of work the job carries out.
	Type constants.JobType `json:"type" validate:"required"`
	// The kind of resource the job operates on, as an object-type value (e.g. `product`).
	//
	// `type` names the verb — what the job does — and this names the subject, so a job that produced no results still says what it was for.
	ResourceType *constants.ObjectType `json:"resource_type"`
	// How far the job has got.
	//
	// `completed` means the work was processed, not that every row succeeded — read each entry's own `status` in `results`.
	Status constants.JobStatus `json:"status" validate:"required"`
	// The actor who requested the work.
	CreatedBy *Actor `json:"created_by" expandable:"true"`
	// One entry per submitted row, saying what became of it. A bulk create records these when it accepts the request, so they stay provisional until `status` is `completed`.
	//
	// `page_info.has_next_page` is true when the job produced more rows than it records.
	Results *List[JobResult] `json:"results"`
	// The failure that sank the job as a whole, in the same shape a synchronous error
	// response carries.
	//
	// A row rejected on its own merits reports its failure on its own entry in `results` instead, so this stays null even when some rows failed.
	Error *apierror.ResponseError `json:"error"`
	// Where a completed export job's file can be downloaded.
	//
	// Null on every other job, and returned only to a caller asking for JSON — otherwise retrieving the job redirects to it.
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
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=job_export"`
	// Presigned link to the file, valid for five minutes.
	//
	// If the link has expired, read the job again for a fresh one.
	URL string `json:"url" validate:"required" sensitive:"true"`
}

// Accounts for one row of the request: the resource it produced, or the error it was
// rejected with. Every submitted row lands in exactly one of these once the job completes.
type JobResult struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=job_result"`
	// Zero-based row of the request this result names.
	Index int `json:"index"`
	// What became of the row.
	//
	// - `created`: the row produced a new resource.
	// - `updated`: the row updated an existing resource.
	// - `failed`: the row was rejected and wrote nothing.
	Status constants.JobResultStatus `json:"status" validate:"required"`
	// The resource the row produced. Null when the row failed.
	Resource *Entity `json:"resource"`
	// Resources produced as a side effect to the row's primary operation.
	//
	// For example, when creating a production run, several batch records may also be created.
	SubResources *List[Entity] `json:"sub_resources"`
	// Why the row was rejected, in the same shape a synchronous error response carries.
	//
	// Null unless `status` is `failed`.
	Error *apierror.ResponseError `json:"error"`
}

// NewJobExport builds the download reference a completed export job carries.
func NewJobExport(url string) *JobExport {
	return &JobExport{Object: constants.ObjectTypeJobExport, URL: url}
}

var SampleJob = &Job{
	ID:           SampleJobID,
	Object:       constants.ObjectTypeJob,
	Type:         constants.JobTypeBulkCreate,
	ResourceType: new(constants.ObjectTypeProductionRun),
	Status:       constants.JobStatusCompleted,
	Results: NewList([]JobResult{{
		Object:   constants.ObjectTypeJobResult,
		Index:    0,
		Status:   constants.JobResultStatusCreated,
		Resource: NewEntity(SampleProductionRunID, constants.ObjectTypeProductionRun, nil, nil),
		SubResources: NewList([]Entity{
			*NewEntity(SampleBatchID, constants.ObjectTypeBatch, nil, nil),
		}, PageInfo{}),
	}}, PageInfo{}),
	StartedAt:   timeutil.TimestampToTimePtr(sampleCreatedAtTimestamp),
	CompletedAt: timeutil.TimestampToTimePtr(sampleCreatedAtTimestamp),
	CreatedAt:   timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:   timeutil.TimestampToTime(sampleCreatedAtTimestamp),
}

func (*Job) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleJob)
}

var SampleJobResult = &SampleJob.Results.Data[0]

func (*JobResult) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleJobResult)
}

var SampleJobExport = NewJobExport("https://files.augno.com/exports/" + SampleAccountID + "/production_runs/" + SampleJobID + "/production_runs_20260817.xlsx?X-Amz-Signature=example")

func (*JobExport) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleJobExport)
}
