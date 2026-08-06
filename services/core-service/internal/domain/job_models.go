package domain

import (
	"encoding/json"
	"time"

	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

type Job struct {
	ID                string
	Type              constants.JobType
	AccountID         *string
	CreatedByID       *string
	CreatedByName     *string
	CreatedByUsername *string
	CreatedByEmail    *string
	JobItems          json.RawMessage
	Results           []RowResult
	Errors            []apierror.RowError
	ErrorSummary      *string
	StartedAt         *time.Time
	CompletedAt       *time.Time
	FailedAt          *time.Time
	CancelledAt       *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// names what one row of a bulk request produced
type RowResult struct {
	Index          int
	ID             string
	Action         constants.JobResultAction
	SubResourceIDs []string
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
	Type        constants.JobType
	JobItems    json.RawMessage
	CreatedByID *string
	Results     []RowResult
}

type CreateJobRepositoryParams struct {
	JobID       string
	JobItems    json.RawMessage
	Type        constants.JobType
	AccountID   string
	CreatedByID *string
	Results     []RowResult
}

type UpdateJobServiceParams struct {
	JobID        string
	Status       constants.JobStatus
	Results      []RowResult
	Errors       []apierror.RowError
	ErrorSummary *string
}

type UpdateJobRepositoryParams struct {
	JobID        string
	AccountID    string
	Results      []RowResult
	Errors       []apierror.RowError
	ErrorSummary *string
	StartedAt    *time.Time
	CompletedAt  *time.Time
	FailedAt     *time.Time
	CancelledAt  *time.Time
}

type StartJobParams struct {
	JobID string
}

type CompleteJobParams struct {
	JobID   string
	Results []RowResult
	Errors  []apierror.RowError
}

type FailJobParams struct {
	JobID  string
	ApiErr *apierror.APIError
}

type CancelJobParams struct {
	JobID string
}
