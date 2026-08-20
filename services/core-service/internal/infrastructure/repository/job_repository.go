package repository

import (
	"context"
	gosql "database/sql"
	"encoding/json"
	"fmt"
	"sort"
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

// objectTypePtr narrows an object type to the optional string the column stores: the
// zero value means the job names no resource kind, which is a NULL rather than an "".
func objectTypePtr(t constants.ObjectType) *string {
	if t == "" {
		return nil
	}
	s := string(t)
	return &s
}

func toNullTime(t *time.Time) gosql.NullTime {
	if t == nil {
		return gosql.NullTime{}
	}
	return gosql.NullTime{Time: *t, Valid: true}
}

// job.results and job.error are nullable json columns holding domain values. Encoding
// them is this layer's job: the JSON is how the column stores them, not something the
// domain or the services above it should know about. They read back as
// db.NullableRawMessage because json.RawMessage cannot scan NULL.
//
// A NULL results column decodes to a nil slice and a nil slice encodes back to NULL,
// while an empty-but-present list round-trips as `[]` — that is what separates a job
// that has recorded no results yet from one that ran and wrote nothing.

// storedResults is the results column's shape. It wraps the rows rather than being a
// bare array so the record can also say it was trimmed, which a bare array cannot.
type storedResults struct {
	Rows      []storedRowResult `json:"rows"`
	Truncated bool              `json:"truncated"`
}

type storedRowResult struct {
	Index        int                       `json:"index"`
	Status       constants.JobResultStatus `json:"status"`
	ResourceType constants.ObjectType      `json:"resource_type,omitempty"`
	ID           string                    `json:"id,omitempty"`
	Name         *string                   `json:"name,omitempty"`
	SubResources []storedSubResource       `json:"sub_resources,omitempty"`
	Error        *apierror.ResponseError   `json:"error,omitempty"`
}

type storedSubResource struct {
	ResourceType constants.ObjectType `json:"resource_type"`
	ID           string               `json:"id"`
	Name         *string              `json:"name,omitempty"`
}

// legacyStoredRowResult reads a row written before a result carried its own status and
// named the object types it produced. Those jobs settle within minutes of being raised,
// so this only has to cover the ones in flight across a deploy — delete it, along with
// the errors column it pairs with, once none predating the merge can still be polled.
type legacyStoredRowResult struct {
	Index          int                       `json:"Index"`
	ID             string                    `json:"ID"`
	Action         constants.JobResultStatus `json:"Action"`
	SubResourceIDs []string                  `json:"SubResourceIDs"`
}

type legacyStoredRowError struct {
	Index *int                   `json:"index"`
	Error apierror.ResponseError `json:"error"`
}

func decodeJSON[T any](n db.NullableRawMessage, column string) (*T, *apierror.APIError) {
	if len(n) == 0 {
		return nil, nil
	}
	var out T
	if err := json.Unmarshal(n, &out); err != nil {
		return nil, apierror.NewInternalError(err, fmt.Sprintf("Failed to decode job %s.", column))
	}
	return &out, nil
}

func encodeJSON[T any](v *T, column string) (db.NullableRawMessage, *apierror.APIError) {
	if v == nil {
		return nil, nil
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, apierror.NewInternalError(err, fmt.Sprintf("Failed to encode job %s.", column))
	}
	return db.NullableRawMessage(raw), nil
}

// isJSONArray reports whether the stored results predate the wrapper — a bare array is
// the legacy shape. See legacyStoredRowResult.
func isJSONArray(n db.NullableRawMessage) bool {
	for _, b := range n {
		switch b {
		case ' ', '\t', '\n', '\r':
			continue
		default:
			return b == '['
		}
	}
	return false
}

func encodeResults(results []domain.RowResult, truncated bool) (db.NullableRawMessage, *apierror.APIError) {
	if results == nil {
		return nil, nil
	}
	rows := make([]storedRowResult, len(results))
	for i, r := range results {
		subs := make([]storedSubResource, len(r.SubResources))
		for j, s := range r.SubResources {
			subs[j] = storedSubResource{ResourceType: s.ResourceType, ID: s.ID, Name: s.Name}
		}
		rows[i] = storedRowResult{
			Index:        r.Index,
			Status:       r.Status,
			ResourceType: r.ResourceType,
			ID:           r.ID,
			Name:         r.Name,
			SubResources: subs,
			Error:        r.Error,
		}
	}
	return encodeJSON(&storedResults{Rows: rows, Truncated: truncated}, "results")
}

// decodeResults reads the row outcomes, folding in a legacy job's separate errors column
// so a job written before the merge still reports one entry per submitted row. The
// whole-job failure it may carry — the legacy entry with no index — is returned
// separately, since that now settles on the job rather than among its rows.
func decodeResults(rawResults, rawErrors db.NullableRawMessage, resourceType constants.ObjectType) ([]domain.RowResult, bool, *apierror.ResponseError, *apierror.APIError) {
	if isJSONArray(rawResults) {
		return decodeLegacyResults(rawResults, rawErrors, resourceType)
	}

	stored, apiErr := decodeJSON[storedResults](rawResults, "results")
	if apiErr != nil {
		return nil, false, nil, apiErr
	}
	if stored == nil {
		return nil, false, nil, nil
	}

	out := make([]domain.RowResult, len(stored.Rows))
	for i, r := range stored.Rows {
		subs := make([]domain.SubResourceRef, len(r.SubResources))
		for j, s := range r.SubResources {
			subs[j] = domain.SubResourceRef{ResourceType: s.ResourceType, ID: s.ID, Name: s.Name}
		}
		out[i] = domain.RowResult{
			Index:        r.Index,
			Status:       r.Status,
			ResourceType: r.ResourceType,
			ID:           r.ID,
			Name:         r.Name,
			SubResources: subs,
			Error:        r.Error,
		}
	}
	return out, stored.Truncated, nil, nil
}

func decodeLegacyResults(rawResults, rawErrors db.NullableRawMessage, resourceType constants.ObjectType) ([]domain.RowResult, bool, *apierror.ResponseError, *apierror.APIError) {
	legacy, apiErr := decodeJSON[[]legacyStoredRowResult](rawResults, "results")
	if apiErr != nil {
		return nil, false, nil, apiErr
	}

	var out []domain.RowResult
	if legacy != nil {
		out = make([]domain.RowResult, 0, len(*legacy))
		for _, r := range *legacy {
			out = append(out, domain.RowResult{
				Index:        r.Index,
				Status:       r.Action,
				ResourceType: resourceType,
				ID:           r.ID,
				SubResources: domain.NewSubResourceRefs(resourceType, r.SubResourceIDs),
			})
		}
	}

	legacyErrs, apiErr := decodeJSON[[]legacyStoredRowError](rawErrors, "errors")
	if apiErr != nil {
		return nil, false, nil, apiErr
	}
	var jobError *apierror.ResponseError
	if legacyErrs != nil {
		for _, e := range *legacyErrs {
			rowErr := e.Error
			if e.Index == nil {
				jobError = &rowErr
				continue
			}
			out = append(out, domain.RowResult{
				Index:  *e.Index,
				Status: constants.JobResultStatusFailed,
				Error:  &rowErr,
			})
		}
		sort.SliceStable(out, func(i, j int) bool { return out[i].Index < out[j].Index })
	}

	return out, false, jobError, nil
}

// legacyJobTypes maps the separator-less spellings the type column held before the enum
// was made snake_case. The stored values are backfilled on deploy, so this only covers
// rows written in the window before it runs — delete it with the backfill.
var legacyJobTypes = map[string]constants.JobType{
	"bulkcreate": constants.JobTypeBulkCreate,
	"bulkupsert": constants.JobTypeBulkUpsert,
}

func mapGetJobRow(row sqlc.GetJobRow) (*domain.Job, *apierror.APIError) {
	jobType := constants.JobType(row.Type)
	if legacy, ok := legacyJobTypes[row.Type]; ok {
		jobType = legacy
	}
	if !jobType.IsValid() {
		jobType = ""
	}

	resourceType := constants.ObjectType(row.ResourceType.String)

	results, truncated, legacyJobError, apiErr := decodeResults(row.Results, row.Errors, resourceType)
	if apiErr != nil {
		return nil, apiErr
	}

	jobError, apiErr := decodeJSON[apierror.ResponseError](row.Error, "error")
	if apiErr != nil {
		return nil, apiErr
	}
	if jobError == nil {
		jobError = legacyJobError
	}

	return &domain.Job{
		ID:               row.ID,
		Type:             jobType,
		ResourceType:     resourceType,
		AccountID:        nullStringPtr(row.AccountID),
		CreatedByID:      nullStringPtr(row.CreatedBy),
		JobItems:         row.JobItems,
		Results:          results,
		ResultsTruncated: truncated,
		Error:            jobError,
		StartedAt:        nullTimePtr(row.StartedAt),
		CompletedAt:      nullTimePtr(row.CompletedAt),
		FailedAt:         nullTimePtr(row.FailedAt),
		CancelledAt:      nullTimePtr(row.CancelledAt),
		CreatedAt:        row.CreatedAt,
		UpdatedAt:        row.UpdatedAt,
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

	results, apiErr := encodeResults(params.Results, false)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	err := r.queries.InsertJob(ctx, sqlc.InsertJobParams{
		ID:           params.JobID,
		AccountID:    toNullString(&params.AccountID),
		Type:         string(params.Type),
		ResourceType: toNullString(objectTypePtr(params.ResourceType)),
		CreatedBy:    toNullString(params.CreatedByID),
		JobItems:     params.JobItems,
		Results:      results,
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

	results, apiErr := encodeResults(params.Results, params.ResultsTruncated)
	if apiErr != nil {
		return 0, tracing.Trace(span, apiErr)
	}
	jobError, apiErr := encodeJSON(params.Error, "error")
	if apiErr != nil {
		return 0, tracing.Trace(span, apiErr)
	}

	rows, err := r.queries.UpdateJob(ctx, sqlc.UpdateJobParams{
		ID:          params.JobID,
		AccountID:   toNullString(&params.AccountID),
		Results:     results,
		Error:       jobError,
		StartedAt:   toNullTime(params.StartedAt),
		CompletedAt: toNullTime(params.CompletedAt),
		FailedAt:    toNullTime(params.FailedAt),
		CancelledAt: toNullTime(params.CancelledAt),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return 0, tracing.Trace(span, apiErr)
	}

	return rows, nil
}
