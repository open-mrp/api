package apiresource

import (
	"github.com/open-mrp/api/shared/constants"
	apiexample "github.com/open-mrp/apikit/example"
	"github.com/open-mrp/apikit/object"
	"github.com/open-mrp/apikit/timeutil"
)

const SampleAsyncJobID = "jb_grz7cdpnz8jr"

// Records a piece of work the API accepted and carries out in the background. Endpoints
// answering `202 Accepted` point at one with a `Location` header; poll it for the outcome.
type AsyncJob struct {
	object.AsyncJob[Entity]
	// The actor who requested the work.
	CreatedBy *Actor `json:"created_by" expandable:"true"`
	// Where a completed export job's file can be downloaded.
	//
	// Null on every other job, and returned only to a caller asking for JSON — otherwise retrieving the job redirects to it.
	Export *AsyncJobExport `json:"export"`
}

// AsyncJobResult accounts for one row of the request: the resource it produced, or the error it was rejected with.
type AsyncJobResult = object.AsyncJobResult[Entity]

// Points a completed export job at the file it produced.
type AsyncJobExport struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=async_job_export"`
	// Presigned link to the file, valid for five minutes.
	//
	// If the link has expired, read the job again for a fresh one.
	URL string `json:"url" validate:"required" sensitive:"true"`
}

// NewAsyncJobExport builds the download reference a completed export job carries.
func NewAsyncJobExport(url string) *AsyncJobExport {
	return &AsyncJobExport{Object: constants.ObjectTypeAsyncJobExport, URL: url}
}

var SampleAsyncJob = &AsyncJob{AsyncJob: object.AsyncJob[Entity]{
	ID:           SampleAsyncJobID,
	Object:       object.TypeAsyncJob,
	Type:         string(constants.JobTypeBulkCreate),
	ResourceType: new(constants.ObjectTypeProductionRun),
	Status:       object.AsyncJobStatusCompleted,
	Results: NewList([]AsyncJobResult{{
		Object:   object.TypeAsyncJobResult,
		Index:    0,
		Status:   object.AsyncJobResultCreated,
		Resource: NewEntity(SampleProductionRunID, constants.ObjectTypeProductionRun, nil, nil),
		SubResources: NewList([]Entity{
			*NewEntity(SampleBatchID, constants.ObjectTypeBatch, nil, nil),
		}, PageInfo{}),
	}}, PageInfo{}),
	StartedAt:   timeutil.TimestampToTimePtr(sampleCreatedAtTimestamp),
	CompletedAt: timeutil.TimestampToTimePtr(sampleCreatedAtTimestamp),
	CreatedAt:   timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:   timeutil.TimestampToTime(sampleCreatedAtTimestamp),
}}

func (*AsyncJob) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleAsyncJob)
}

var SampleAsyncJobExport = NewAsyncJobExport("https://files.openmrp.ai/exports/" + SampleAccountID + "/production_runs/" + SampleAsyncJobID + "/production_runs_20260817.xlsx?X-Amz-Signature=example")

func (*AsyncJobExport) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleAsyncJobExport)
}
