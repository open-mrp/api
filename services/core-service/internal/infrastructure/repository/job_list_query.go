package repository

import (
	"context"
	"strings"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/db"
	"github.com/open-mrp/apikit/pagination"
	"github.com/open-mrp/apikit/tracing"
)

// jobListColumns is GetJob's select list, in its order. Only an export's payload is read: its download link is
// signed from it, and no other job's payload is part of a listing.
const jobListColumns = `j.job_id, j.type, j.resource_type, j.account_id, j.created_by, ` +
	`IF(j.type = 'export', j.job_items, NULL), IF(j.type = 'export', j.job_items_key, NULL), ` +
	`j.results, j.error, j.errors, j.started_at, j.completed_at, j.failed_at, j.cancelled_at, j.created_at, j.updated_at`

// jobStatusPredicates mirror domain.Job.Status: a later timestamp only decides the status while every earlier one is unset.
var jobStatusPredicates = map[constants.JobStatus]string{
	constants.JobStatusCanceled:  "j.cancelled_at IS NOT NULL",
	constants.JobStatusCompleted: "(j.cancelled_at IS NULL AND j.completed_at IS NOT NULL)",
	constants.JobStatusFailed:    "(j.cancelled_at IS NULL AND j.completed_at IS NULL AND j.failed_at IS NOT NULL)",
	constants.JobStatusRunning:   "(j.cancelled_at IS NULL AND j.completed_at IS NULL AND j.failed_at IS NULL AND j.started_at IS NOT NULL)",
	constants.JobStatusQueued:    "(j.cancelled_at IS NULL AND j.completed_at IS NULL AND j.failed_at IS NULL AND j.started_at IS NULL)",
}

// buildJobListQuery selects a page of the account's jobs. Only the filters the caller sent are emitted. Forward pages
// read older (DESC), backward newer (ASC); the key yields the account's jobs in either order without a sort.
func buildJobListQuery(params domain.ListJobsRepositoryParams, dir pagination.Direction, cursorCreatedAt *time.Time, cursorID string, limit int32) (string, []any) {
	where := []string{"j.account_id = ?"}
	args := []any{params.AccountID}

	if params.CreatedByID != nil {
		where, args = append(where, "j.created_by = ?"), append(args, *params.CreatedByID)
	}
	if len(params.Types) > 0 {
		var stored []any
		for _, t := range params.Types {
			stored = append(stored, string(t))
			for legacy, current := range legacyJobTypes {
				if current == t {
					stored = append(stored, legacy)
				}
			}
		}
		where = append(where, "j.type IN ("+placeholders(len(stored))+")")
		args = append(args, stored...)
	}
	if len(params.Statuses) > 0 {
		var arms []string
		seen := map[constants.JobStatus]bool{}
		for _, s := range params.Statuses {
			if p, ok := jobStatusPredicates[s]; ok && !seen[s] {
				seen[s] = true
				arms = append(arms, p)
			}
		}
		if len(arms) > 0 {
			where = append(where, "("+strings.Join(arms, " OR ")+")")
		}
	}
	order := " ORDER BY j.created_at DESC, j.job_id DESC"
	if cursorCreatedAt != nil {
		if dir == pagination.DirectionBackward {
			where = append(where, "(j.created_at > ? OR (j.created_at = ? AND j.job_id > ?))")
			order = " ORDER BY j.created_at ASC, j.job_id ASC"
		} else {
			where = append(where, "(j.created_at < ? OR (j.created_at = ? AND j.job_id < ?))")
		}
		args = append(args, *cursorCreatedAt, *cursorCreatedAt, cursorID)
	}
	args = append(args, limit)
	return "SELECT " + jobListColumns + " FROM job j FORCE INDEX (job_account_created_idx) WHERE " +
		strings.Join(where, " AND ") + order + " LIMIT ?", args
}

func (r *jobRepoImpl) List(ctx context.Context, params domain.ListJobsRepositoryParams) (*domain.ListJobsResult, *apierror.APIError) {
	ctx, span := jobRepoTracer.Start(ctx, "repository.job.list")
	defer span.End()

	var cursorDir *pagination.Direction
	var cursorCreatedAt *time.Time
	var cursorID string
	if params.Cursor != nil {
		cur, err := pagination.DecodeStringCursor(*params.Cursor)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewParameterInvalidError("cursor", "Invalid pagination cursor."))
		}
		cursorDir, cursorCreatedAt, cursorID = &cur.Direction, &cur.OccurredAt, cur.ID
	}
	dir := pagination.DirectionForward
	if cursorDir != nil {
		dir = *cursorDir
	}

	query, args := buildJobListQuery(params, dir, cursorCreatedAt, cursorID, params.Limit+1)
	rows, err := r.queries.DB().QueryContext(ctx, query, args...)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	defer rows.Close()

	var jobs []*domain.Job
	for rows.Next() {
		var row sqlc.GetJobRow
		if err := rows.Scan(&row.ID, &row.Type, &row.ResourceType, &row.AccountID, &row.CreatedBy, &row.JobItems,
			&row.JobItemsKey, &row.Results, &row.Error, &row.Errors, &row.StartedAt, &row.CompletedAt, &row.FailedAt,
			&row.CancelledAt, &row.CreatedAt, &row.UpdatedAt); err != nil {
			return nil, tracing.Trace(span, db.MapSQLError(err))
		}
		job, apiErr := mapGetJobRow(row)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		jobs = append(jobs, job)
	}
	if apiErr := db.MapSQLError(rows.Err()); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	page, info := pagination.BuildPageString(jobs, params.Limit, cursorDir,
		func(j *domain.Job) time.Time { return j.CreatedAt }, func(j *domain.Job) string { return j.ID })
	return &domain.ListJobsResult{Jobs: page, PageInfo: info}, nil
}
