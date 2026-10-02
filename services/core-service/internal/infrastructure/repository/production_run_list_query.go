package repository

import (
	"context"
	gosql "database/sql"
	"strings"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/pagination"
	"github.com/open-mrp/api/shared/safeconv"
	"github.com/open-mrp/api/shared/tracing"
)

// productionRunListIndex yields an account's runs in list order.
const productionRunListIndex = "production_run_account_created_idx"

// productionRunCountedRatio sets how many runs, in pages, make a machine common. A var so a test can
// reach the common path on a small corpus.
var productionRunCountedRatio = 40

// productionRunListColumns is the run summary scanProductionRunSummaries reads. A run's batch count is
// counted for the page's runs only; joining batches before the page is cut counted every run's.
const productionRunListColumns = `
	pr.id,
	pr.number,
	pr.responsible_user_id,
	au.id,
	COALESCE(u.name, au.id, ''),
	au.status_code,
	au.created_at,
	au.updated_at,
	pr.started_at,
	pr.completed_at,
	pr.created_at,
	pr.updated_at,
	(SELECT COUNT(*) FROM batch b WHERE b.production_run_id = pr.id AND b.account_id = pr.account_id)`

// productionRunListJoins name the responsible user, stored as either an account_user id or a legacy
// user id; both are matched, scoped to the run's account.
const productionRunListJoins = `
LEFT JOIN account_user au ON au.account_id = pr.account_id AND (au.id = pr.responsible_user_id OR au.user_id = pr.responsible_user_id)
LEFT JOIN ` + "`user`" + ` u ON u.id = au.user_id`

func (r *productionRunRepoImpl) List(ctx context.Context, params domain.ListProductionRunsParams) (*domain.ListProductionRunsResult, *apierror.APIError) {
	ctx, span := productionRunRepoTracer.Start(ctx, "repository.production_run.list")
	defer span.End()

	var cur *pagination.StringCursor
	var cursorDir *pagination.Direction
	if params.Cursor != nil {
		decoded, err := pagination.DecodeStringCursor(*params.Cursor)
		if err != nil {
			return nil, apierror.NewValidationErrorWithParam("Invalid pagination cursor.", "cursor")
		}
		cur, cursorDir = &decoded, &decoded.Direction
	}
	empty := func() (*domain.ListProductionRunsResult, *apierror.APIError) {
		result, pageInfo := pagination.BuildPageString([]*domain.ProductionRunSummary{}, params.Limit, cursorDir, productionRunSummaryCreatedAt, productionRunSummaryID)
		return r.listResultWithSummaries(ctx, span, params.AccountID, result, pageInfo)
	}

	f := &whereClause{}
	f.add("pr.account_id = ?", params.AccountID)
	index := productionRunListIndex

	includeStatusFilter, statusOpen, statusClosed, includeItemFilter, itemIDs, includeMachineFilter, machineIDs := buildProductionRunListFilters(params)
	if includeStatusFilter {
		switch {
		case statusOpen:
			f.add("pr.completed_at IS NULL")
		case statusClosed:
			f.add("pr.completed_at IS NOT NULL")
		default:
			return empty()
		}
	}

	if numberQuery, batchIDQuery := buildProductionRunSearchParams(params.Query); numberQuery.Valid {
		// Runs whose batch ids start with the term are found from the batch key, rather than by scanning
		// every run's batches for one.
		runIDs, err := r.runIDsOf(ctx, `
SELECT DISTINCT production_run_id FROM batch
WHERE account_id = ? AND id LIKE ? AND production_run_id IS NOT NULL`, params.AccountID, batchIDQuery.String)
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		if len(runIDs) == 0 {
			f.add("pr.number LIKE ?", numberQuery.String)
		} else {
			f.add("(pr.number LIKE ? OR pr.id IN ("+placeholders(len(runIDs))+"))", append([]any{numberQuery.String}, stringArgs(runIDs)...)...)
		}
	}

	if includeItemFilter {
		// A probe of the run's (account, run, item) key per run walked.
		f.add(`EXISTS (SELECT 1 FROM batch b2 WHERE b2.account_id = pr.account_id AND b2.production_run_id = pr.id
			AND b2.item_id IN (`+placeholders(len(itemIDs))+`))`, stringArgs(itemIDs)...)
	}

	if includeMachineFilter {
		// No batch key leads with the run and the machine, so the machine's batches are found first.
		runIDs, err := r.runIDsOf(ctx, `
SELECT DISTINCT b3.production_run_id FROM _batches_machines bm JOIN batch b3 ON b3.id = bm.A
WHERE bm.B IN (`+placeholders(len(machineIDs))+`) AND b3.account_id = ? AND b3.production_run_id IS NOT NULL`,
			append(stringArgs(machineIDs), params.AccountID)...)
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		switch {
		case len(runIDs) == 0:
			return empty()
		case len(runIDs) <= productionRunCountedRatio*int(params.Limit+1):
			// Few enough to read by id and sort. Walking the list key with them as an IN list instead, the
			// planner may scan the key from the account's far end rather than range it from the cursor.
			f.in("pr.id", runIDs)
			index = "PRIMARY"
		default:
			// Common enough that walking the list key in order finds a page quickly; each run walked is
			// probed through its own batches.
			f.add(`EXISTS (SELECT 1 FROM batch b4 JOIN _batches_machines bm4 ON bm4.A = b4.id
			WHERE b4.account_id = pr.account_id AND b4.production_run_id = pr.id
			AND bm4.B IN (`+placeholders(len(machineIDs))+`))`, stringArgs(machineIDs)...)
		}
	}

	if startDate := parseDateString(params.StartDate); startDate.Valid {
		f.add("pr.created_at >= ?", startDate.Time)
	}
	if endDate := parseDateString(params.EndDate); endDate.Valid {
		f.add("pr.created_at <= ?", endDate.Time)
	}

	orderBy := "pr.created_at DESC, pr.id DESC"
	if cur != nil {
		if cur.Direction == pagination.DirectionBackward {
			f.add("(pr.created_at > ? OR (pr.created_at = ? AND pr.id > ?))", cur.OccurredAt, cur.OccurredAt, cur.ID)
			orderBy = "pr.created_at ASC, pr.id ASC"
		} else {
			f.add("(pr.created_at < ? OR (pr.created_at = ? AND pr.id < ?))", cur.OccurredAt, cur.OccurredAt, cur.ID)
		}
	}

	query := "SELECT" + productionRunListColumns +
		"\nFROM (SELECT pr.id FROM production_run pr FORCE INDEX (" + index + ")" +
		"\nWHERE " + strings.Join(f.where, "\nAND ") +
		"\nORDER BY " + orderBy + "\nLIMIT ?) page" +
		"\nJOIN production_run pr ON pr.id = page.id" + productionRunListJoins +
		"\nORDER BY " + orderBy
	rows, err := r.queries.DB().QueryContext(ctx, query, append(f.args, params.Limit+1)...)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	defer func() { _ = rows.Close() }()
	runs, err := scanProductionRunSummaries(rows)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	result, pageInfo := pagination.BuildPageString(runs, params.Limit, cursorDir, productionRunSummaryCreatedAt, productionRunSummaryID)
	return r.listResultWithSummaries(ctx, span, params.AccountID, result, pageInfo)
}

// whereClause collects a statement's predicates and their bind args.
type whereClause struct {
	where []string
	args  []any
}

func (w *whereClause) add(clause string, args ...any) {
	w.where = append(w.where, clause)
	w.args = append(w.args, args...)
}

func (w *whereClause) in(column string, values []string) {
	if len(values) > 0 {
		w.add(column+" IN ("+placeholders(len(values))+")", stringArgs(values)...)
	}
}

// runIDsOf runs a query returning production run ids.
func (r *productionRunRepoImpl) runIDsOf(ctx context.Context, query string, args ...any) ([]string, error) {
	rows, err := r.queries.DB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func scanProductionRunSummaries(rows *gosql.Rows) ([]*domain.ProductionRunSummary, error) {
	runs := []*domain.ProductionRunSummary{}
	for rows.Next() {
		var s domain.ProductionRunSummary
		var responsibleUserID, responsibleUserName string
		var accountUserID, statusCode gosql.NullString
		var userCreatedAt, userUpdatedAt, startedAt, completedAt gosql.NullTime
		var batchCount int64
		if err := rows.Scan(&s.ID, &s.Number, &responsibleUserID, &accountUserID, &responsibleUserName, &statusCode,
			&userCreatedAt, &userUpdatedAt, &startedAt, &completedAt, &s.CreatedAt, &s.UpdatedAt, &batchCount); err != nil {
			return nil, err
		}
		s.ResponsibleUserID = resolvedResponsibleUserID(accountUserID, responsibleUserID)
		s.BatchCount = safeconv.Int64ToInt32(batchCount)
		if responsibleUserName != "" {
			s.ResponsibleUserName = &responsibleUserName
		}
		if statusCode.Valid {
			s.ResponsibleUserStatusCode = &statusCode.String
		}
		if userCreatedAt.Valid {
			s.ResponsibleUserCreatedAt = &userCreatedAt.Time
		}
		if userUpdatedAt.Valid {
			s.ResponsibleUserUpdatedAt = &userUpdatedAt.Time
		}
		if startedAt.Valid {
			s.StartedAt = &startedAt.Time
		}
		if completedAt.Valid {
			s.CompletedAt = &completedAt.Time
		}
		runs = append(runs, &s)
	}
	return runs, rows.Err()
}
