package repository

import (
	"context"
	gosql "database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/tracing"
)

var jobRepoTracer = tracing.GetTracer("core-service.job_repository")

type jobRepoImpl struct {
	queries *sqlc.Queries
}

func NewJobRepo(queries *sqlc.Queries) domain.JobRepo {
	return &jobRepoImpl{queries: queries}
}

func toNullTime(t *time.Time) gosql.NullTime {
	if t == nil {
		return gosql.NullTime{}
	}
	return gosql.NullTime{Time: *t, Valid: true}
}

// job.results and job.errors are nullable json columns holding a list of domain
// values. Encoding them is this layer's job: the JSON is how the column stores the
// list, not something the domain or the services above it should know about. They
// read back as db.NullableRawMessage because json.RawMessage cannot scan NULL.
//
// A NULL column decodes to a nil slice and a nil slice encodes back to NULL, while an
// empty-but-present list round-trips as `[]` — that is what separates a job that has
// recorded no results yet from one that ran and wrote nothing.
func decodeJSONList[T any](n db.NullableRawMessage, column string) ([]T, *apierror.APIError) {
	if len(n) == 0 {
		return nil, nil
	}
	var out []T
	if err := json.Unmarshal(n, &out); err != nil {
		return nil, apierror.NewInternalError(err, fmt.Sprintf("Failed to decode job %s.", column))
	}
	return out, nil
}

func encodeJSONList[T any](list []T, column string) (db.NullableRawMessage, *apierror.APIError) {
	if list == nil {
		return nil, nil
	}
	raw, err := json.Marshal(list)
	if err != nil {
		return nil, apierror.NewInternalError(err, fmt.Sprintf("Failed to encode job %s.", column))
	}
	return db.NullableRawMessage(raw), nil
}

func mapGetJobRow(row sqlc.GetJobRow) (*domain.Job, *apierror.APIError) {
	jobType := constants.JobType(row.Type)
	if !jobType.IsValid() {
		jobType = ""
	}

	results, apiErr := decodeJSONList[domain.RowResult](row.Results, "results")
	if apiErr != nil {
		return nil, apiErr
	}
	errs, apiErr := decodeJSONList[apierror.RowError](row.Errors, "errors")
	if apiErr != nil {
		return nil, apiErr
	}

	return &domain.Job{
		ID:                row.ID,
		Type:              jobType,
		AccountID:         nullStringPtr(row.AccountID),
		CreatedByID:       nullStringPtr(row.CreatedBy),
		CreatedByName:     nullStringPtr(row.CreatedByName),
		CreatedByUsername: nullStringPtr(row.CreatedByUsername),
		CreatedByEmail:    nullStringPtr(row.CreatedByEmail),
		JobItems:          row.JobItems,
		Results:           results,
		Errors:            errs,
		ErrorSummary:      nullStringPtr(row.ErrorSummary),
		StartedAt:         nullTimePtr(row.StartedAt),
		CompletedAt:       nullTimePtr(row.CompletedAt),
		FailedAt:          nullTimePtr(row.FailedAt),
		CancelledAt:       nullTimePtr(row.CancelledAt),
		CreatedAt:         row.CreatedAt,
		UpdatedAt:         row.UpdatedAt,
	}, nil
}

func (r *jobRepoImpl) Get(ctx context.Context, jobID string, accountID string) (*domain.Job, *apierror.APIError) {
	ctx, span := jobRepoTracer.Start(ctx, "repository.job.get")
	defer span.End()

	row, err := r.queries.GetJob(ctx, sqlc.GetJobParams{
		ID:        jobID,
		AccountID: toNullString(&accountID),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	job, apiErr := mapGetJobRow(row)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return job, nil
}

func (r *jobRepoImpl) Create(ctx context.Context, params domain.CreateJobRepositoryParams) *apierror.APIError {
	ctx, span := jobRepoTracer.Start(ctx, "repository.job.create")
	defer span.End()

	results, apiErr := encodeJSONList(params.Results, "results")
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	err := r.queries.InsertJob(ctx, sqlc.InsertJobParams{
		ID:        params.JobID,
		AccountID: toNullString(&params.AccountID),
		Type:      string(params.Type),
		CreatedBy: toNullString(params.CreatedByID),
		JobItems:  params.JobItems,
		Results:   results,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

// Update patches the job. Nil fields are left as stored: the query coalesces
// every assignment onto the current value, so callers only send what changed.
// Update applies a lifecycle transition and returns the number of rows it changed.
// The query guards on the terminal timestamps, so a job that already settled matches
// no row and the count is zero — the caller treats that as "already settled" rather
// than an error, which is what serializes a client cancel against the worker's
// completion.
func (r *jobRepoImpl) Update(ctx context.Context, params domain.UpdateJobRepositoryParams) (int64, *apierror.APIError) {
	ctx, span := jobRepoTracer.Start(ctx, "repository.job.update")
	defer span.End()

	results, apiErr := encodeJSONList(params.Results, "results")
	if apiErr != nil {
		return 0, tracing.Trace(span, apiErr)
	}
	errs, apiErr := encodeJSONList(params.Errors, "errors")
	if apiErr != nil {
		return 0, tracing.Trace(span, apiErr)
	}

	rows, err := r.queries.UpdateJob(ctx, sqlc.UpdateJobParams{
		ID:           params.JobID,
		AccountID:    toNullString(&params.AccountID),
		Results:      results,
		Errors:       errs,
		ErrorSummary: toNullString(params.ErrorSummary),
		StartedAt:    toNullTime(params.StartedAt),
		CompletedAt:  toNullTime(params.CompletedAt),
		FailedAt:     toNullTime(params.FailedAt),
		CancelledAt:  toNullTime(params.CancelledAt),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return 0, tracing.Trace(span, apiErr)
	}

	return rows, nil
}
