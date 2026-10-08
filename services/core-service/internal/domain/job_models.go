package domain

import (
	"encoding/json"
	"time"

	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

type Job struct {
	ID           string
	Type         constants.JobType
	ResourceType constants.ObjectType
	AccountID    *string
	CreatedByID  *string
	// JobItems is the payload the job was raised with, kept off the wire and out of cached
	// responses. A payload placed by JobSvc.StageJobItems may live in object storage under
	// JobItemsKey instead, with JobItems nil; read those through JobSvc.GetJobItems.
	JobItems    json.RawMessage `json:"-"`
	JobItemsKey *string         `json:"-"`
	Results     []RowResult
	// ResultsTruncated reports that the executor produced more rows than Results carries.
	ResultsTruncated bool
	// Error is the failure that sank the job as a whole. A row that failed carries its
	// own on its RowResult instead.
	Error       *apierror.ResponseError
	StartedAt   *time.Time
	CompletedAt *time.Time
	FailedAt    *time.Time
	CancelledAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// names what became of one row of a bulk request: the resource it produced, or the
// error it was rejected with. Exactly one of ID/Error is set, per Status.
type RowResult struct {
	Index        int
	Status       constants.JobResultStatus
	ResourceType constants.ObjectType
	ID           string
	// The resource's human-readable reference — a shipment number, say. Set only where the
	// operation has one to hand, so a client can name what it wrote without a second read.
	Name         *string
	SubResources []SubResourceRef
	Error        *apierror.ResponseError
}

// references one resource produced alongside a result row's own — a production run's
// batches, say.
type SubResourceRef struct {
	ResourceType constants.ObjectType
	ID           string
	Name         *string
}

// NewSubResourceRefs tags a run of ids produced by one row with the object type they
// all share, which is every case that has come up: a row's sub-resources are siblings.
func NewSubResourceRefs(resourceType constants.ObjectType, ids []string) []SubResourceRef {
	if len(ids) == 0 {
		return nil
	}
	refs := make([]SubResourceRef, len(ids))
	for i, id := range ids {
		refs[i] = SubResourceRef{ResourceType: resourceType, ID: id}
	}
	return refs
}

// Failed reports whether the row was rejected rather than written.
func (r RowResult) Failed() bool {
	return r.Status == constants.JobResultStatusFailed
}

func (j *Job) Status() constants.JobStatus {
	switch {
	case j.CancelledAt != nil:
		return constants.JobStatusCancelled
	case j.CompletedAt != nil:
		return constants.JobStatusCompleted
	case j.FailedAt != nil:
		return constants.JobStatusFailed
	case j.StartedAt != nil:
		return constants.JobStatusStarted
	default:
		return constants.JobStatusCreated
	}
}

func (j *Job) IsTerminal() bool {
	switch j.Status() {
	case constants.JobStatusCompleted, constants.JobStatusCancelled:
		return true
	default:
		return false
	}
}

type CreateJobServiceParams struct {
	// JobID is the id StageJobItems reserved; empty generates one.
	JobID        string
	Type         constants.JobType
	ResourceType constants.ObjectType
	// Exactly one of JobItems / JobItemsKey is set, as StageJobItems returned them.
	JobItems    json.RawMessage
	JobItemsKey *string
	CreatedByID *string
	Results     []RowResult
}

// StagedJobItems is a job payload placed before the job row is written: inline in JobItems, or
// already in object storage under JobItemsKey.
type StagedJobItems struct {
	JobID       string
	JobItems    json.RawMessage
	JobItemsKey *string
}

type CreateJobRepositoryParams struct {
	JobID        string
	JobItems     json.RawMessage
	JobItemsKey  *string
	Type         constants.JobType
	ResourceType constants.ObjectType
	AccountID    string
	CreatedByID  *string
	Results      []RowResult
}

type UpdateJobServiceParams struct {
	JobID   string
	Status  constants.JobStatus
	Results []RowResult
	Error   *apierror.ResponseError
}

type UpdateJobRepositoryParams struct {
	JobID            string
	AccountID        string
	Results          []RowResult
	ResultsTruncated bool
	Error            *apierror.ResponseError
	StartedAt        *time.Time
	CompletedAt      *time.Time
	FailedAt         *time.Time
	CancelledAt      *time.Time
}

type StartJobParams struct {
	JobID string
}

type CompleteJobParams struct {
	JobID   string
	Results []RowResult
}

type FailJobParams struct {
	JobID  string
	ApiErr *apierror.APIError
}

type CancelJobParams struct {
	JobID string
}
