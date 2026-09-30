package repository

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/tracing"
)

const salesRollupSyncName = "rollup"

// Rollup grains, as stored in sales_fact_rollup.grain.
const (
	rollupGrainHour  = "hour"
	rollupGrainDay   = "day"
	rollupGrainMonth = "month"
)

// Rollup dimensions, as stored in sales_fact_rollup.dimension.
const (
	rollupDimTotal       = "total"
	rollupDimBuyer       = "buyer"
	rollupDimItem        = "item"
	rollupDimProductLine = "product_line"
	rollupDimSalesRep    = "sales_rep"
	rollupDimDiscount    = "discount"
)

// salesRollupDimension is how one dimension's bucket rows are derived from sales_line_fact (alias f).
type salesRollupDimension struct {
	name string
	// column is the fact column the rows group on; empty for the account-wide total.
	column string
	// perProductLine adds a row per product line beside the all-product-line row. The product line dimension needs none: its own key already is the product line.
	perProductLine bool
	// hourly also keeps hour buckets, which the daily chart regroups into local days.
	hourly bool
}

var salesRollupDimensions = []salesRollupDimension{
	{name: rollupDimTotal, perProductLine: true, hourly: true},
	{name: rollupDimBuyer, column: "f.buyer_account_id", perProductLine: true},
	{name: rollupDimItem, column: "f.item_id", perProductLine: true},
	{name: rollupDimProductLine, column: "f.product_line_id"},
	// Lines without a sales rep or discount belong to no group of those dimensions.
	{name: rollupDimSalesRep, column: "f.sales_rep_id", perProductLine: true},
	{name: rollupDimDiscount, column: "f.order_discount_id", perProductLine: true},
}

const rollupColumns = `account_id, sales_order_type_code, dimension, product_line_key, grain, bucket_start, row_hash, dimension_id, sales_rep_key,
quantity_base, total_invoiced, total_cost, invoice_count, line_count`

// rollupRowHash is the key's stand-in for (dimension_id, sales_rep_key); NUL cannot occur in either, so distinct pairs never concatenate alike.
func rollupRowHash(dimensionID, salesRepKey string) string {
	return "UNHEX(MD5(CONCAT(" + dimensionID + ", CHAR(0), " + salesRepKey + ")))"
}

// rollupDayInsert fills one dimension's buckets of an account's day. A day statement binds (bucket, account, start, end); an hour statement derives each bucket from the line and binds (account, start, end).
type rollupDayInsert struct {
	sql    string
	hourly bool
}

func rollupDayInserts() []rollupDayInsert {
	var out []rollupDayInsert
	for _, d := range salesRollupDimensions {
		for _, byLine := range []bool{false, true} {
			if byLine && !d.perProductLine {
				continue
			}
			for _, grain := range []string{rollupGrainDay, rollupGrainHour} {
				if grain == rollupGrainHour && !d.hourly {
					continue
				}
				out = append(out, rollupDayInsert{sql: rollupInsert(d, byLine, grain), hourly: grain == rollupGrainHour})
			}
		}
	}
	return out
}

func rollupInsert(d salesRollupDimension, byLine bool, grain string) string {
	dimID, lineKey, bucket := "''", "''", "?"
	group := []string{"f.sales_order_type_code", "COALESCE(f.sales_rep_id, '')"}
	if d.column != "" {
		dimID = d.column
		group = append(group, d.column)
	}
	if byLine {
		lineKey = "f.product_line_id"
		group = append(group, lineKey)
	}
	if grain == rollupGrainHour {
		bucket = "DATE_FORMAT(f.invoiced_at, '%Y-%m-%d %H:00:00')"
		group = append(group, bucket)
	}
	where := "f.account_id = ? AND f.invoiced_at >= ? AND f.invoiced_at < ?"
	if d.column != "" {
		where += " AND " + d.column + " IS NOT NULL"
	}
	// The hash is taken outside the aggregate: ONLY_FULL_GROUP_BY rejects an expression over grouped expressions.
	return `INSERT INTO sales_fact_rollup (` + rollupColumns + `)
SELECT g.account_id, g.sales_order_type_code, '` + d.name + `', g.line_key, '` + grain + `', g.bucket, ` + rollupRowHash("g.dim_id", "g.rep_key") + `, g.dim_id, g.rep_key,
g.qty, g.inv, g.cost, g.ic, g.lc
FROM (
  SELECT f.account_id, f.sales_order_type_code, ` + lineKey + ` AS line_key, ` + bucket + ` AS bucket, ` + dimID + ` AS dim_id, COALESCE(f.sales_rep_id, '') AS rep_key,
  SUM(f.quantity_base) AS qty, SUM(f.total_invoiced) AS inv, SUM(f.total_cost) AS cost, COUNT(DISTINCT f.invoice_id) AS ic, COUNT(*) AS lc
  FROM sales_line_fact f WHERE ` + where + `
  GROUP BY f.account_id, ` + strings.Join(group, ", ") + `
) g`
}

var salesRollupDayInserts = rollupDayInserts()

func (r *salesFactRepoImpl) RebuildRollupDay(ctx context.Context, day domain.SalesRollupDay) *apierror.APIError {
	ctx, span := salesFactRepoTracer.Start(ctx, "repository.sales_fact.rebuild_rollup_day")
	defer span.End()

	start := truncateUTCDay(day.Day)
	end := start.AddDate(0, 0, 1)
	return tracing.Trace(span, r.inTx(ctx, func(tx sqlc.DBTX) error {
		if _, err := tx.ExecContext(ctx, `DELETE FROM sales_fact_rollup WHERE account_id = ? AND bucket_start >= ? AND bucket_start < ? AND grain IN ('hour', 'day')`,
			day.AccountID, start, end); err != nil {
			return err
		}
		for _, stmt := range salesRollupDayInserts {
			args := []any{day.AccountID, start, end}
			if !stmt.hourly {
				args = append([]any{start}, args...)
			}
			if _, err := tx.ExecContext(ctx, stmt.sql, args...); err != nil {
				return err
			}
		}
		return nil
	}))
}

func (r *salesFactRepoImpl) RebuildRollupMonth(ctx context.Context, accountID string, month time.Time) *apierror.APIError {
	ctx, span := salesFactRepoTracer.Start(ctx, "repository.sales_fact.rebuild_rollup_month")
	defer span.End()

	start := truncateUTCMonth(month)
	end := start.AddDate(0, 1, 0)
	// An invoice falls on one day, so a month's distinct invoice count is the sum of its days'.
	return tracing.Trace(span, r.inTx(ctx, func(tx sqlc.DBTX) error {
		if _, err := tx.ExecContext(ctx, `DELETE FROM sales_fact_rollup WHERE account_id = ? AND bucket_start = ? AND grain = 'month'`, accountID, start); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, `INSERT INTO sales_fact_rollup (`+rollupColumns+`)
SELECT account_id, sales_order_type_code, dimension, product_line_key, 'month', ?, row_hash, dimension_id, sales_rep_key,
SUM(quantity_base), SUM(total_invoiced), SUM(total_cost), SUM(invoice_count), SUM(line_count)
FROM sales_fact_rollup
WHERE account_id = ? AND bucket_start >= ? AND bucket_start < ? AND grain = 'day'
GROUP BY account_id, sales_order_type_code, dimension, product_line_key, row_hash, dimension_id, sales_rep_key`, start, accountID, start, end)
		return err
	}))
}

// inTx runs fn in a transaction, so a report never reads a bucket half rebuilt.
func (r *salesFactRepoImpl) inTx(ctx context.Context, fn func(tx sqlc.DBTX) error) *apierror.APIError {
	pool, ok := r.queries.DB().(*sql.DB)
	if !ok {
		// Already inside the caller's transaction.
		return db.MapSQLError(fn(r.queries.DB()))
	}
	tx, err := pool.BeginTx(ctx, nil)
	if err != nil {
		return db.MapSQLError(err)
	}
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return db.MapSQLError(err)
	}
	return db.MapSQLError(tx.Commit())
}

func (r *salesFactRepoImpl) NextRollupDay(ctx context.Context, from domain.SalesRollupDay) (*domain.SalesRollupDay, *apierror.APIError) {
	ctx, span := salesFactRepoTracer.Start(ctx, "repository.sales_fact.next_rollup_day")
	defer span.End()

	account, after := from.AccountID, truncateUTCDay(from.Day)
	for {
		if account != "" {
			// Days with facts, and days whose rollups outlived their facts, both need a rebuild.
			first, apiErr := r.earliest(ctx,
				`SELECT invoiced_at FROM sales_line_fact WHERE account_id = ? AND invoiced_at >= ? ORDER BY invoiced_at LIMIT 1`,
				`SELECT bucket_start FROM sales_fact_rollup WHERE account_id = ? AND bucket_start >= ? ORDER BY bucket_start LIMIT 1`,
				account, after)
			if apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			if first != nil {
				return &domain.SalesRollupDay{AccountID: account, Day: truncateUTCDay(*first)}, nil
			}
		}
		next, apiErr := r.nextAccount(ctx, account)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		if next == "" {
			return nil, nil
		}
		account, after = next, salesFactSweepFloor
	}
}

// earliest returns the smaller of two single-timestamp queries' results, each run with args.
func (r *salesFactRepoImpl) earliest(ctx context.Context, q1, q2 string, args ...any) (*time.Time, *apierror.APIError) {
	var out *time.Time
	for _, q := range []string{q1, q2} {
		var t time.Time
		err := r.queries.DB().QueryRowContext(ctx, q, args...).Scan(&t)
		if errors.Is(err, sql.ErrNoRows) {
			continue
		}
		if err != nil {
			return nil, db.MapSQLError(err)
		}
		if out == nil || t.Before(*out) {
			out = &t
		}
	}
	return out, nil
}

func (r *salesFactRepoImpl) nextAccount(ctx context.Context, after string) (string, *apierror.APIError) {
	next := ""
	for _, q := range []string{
		`SELECT account_id FROM sales_line_fact WHERE account_id > ? ORDER BY account_id LIMIT 1`,
		`SELECT account_id FROM sales_fact_rollup WHERE account_id > ? ORDER BY account_id LIMIT 1`,
	} {
		var id string
		err := r.queries.DB().QueryRowContext(ctx, q, after).Scan(&id)
		if errors.Is(err, sql.ErrNoRows) {
			continue
		}
		if err != nil {
			return "", db.MapSQLError(err)
		}
		if next == "" || id < next {
			next = id
		}
	}
	return next, nil
}

func (r *salesFactRepoImpl) GetRollupSync(ctx context.Context) (*domain.SalesRollupSync, *apierror.APIError) {
	ctx, span := salesFactRepoTracer.Start(ctx, "repository.sales_fact.get_rollup_sync")
	defer span.End()

	row, found, err := r.getSalesSync(ctx, salesRollupSyncName)
	if err != nil {
		return nil, tracing.Trace(span, db.MapSQLError(err))
	}
	if !found {
		return &domain.SalesRollupSync{}, nil
	}
	sync := &domain.SalesRollupSync{PassStartedAt: nullTimePtr(row.passStartedAt), LastCompletedAt: nullTimePtr(row.lastCompletedAt)}
	if row.cursorAccountID.Valid && row.cursorDay.Valid {
		sync.Cursor = &domain.SalesRollupDay{AccountID: row.cursorAccountID.String, Day: row.cursorDay.Time.UTC()}
	}
	return sync, nil
}

func (r *salesFactRepoImpl) SaveRollupSync(ctx context.Context, sync domain.SalesRollupSync) *apierror.APIError {
	ctx, span := salesFactRepoTracer.Start(ctx, "repository.sales_fact.save_rollup_sync")
	defer span.End()

	row := salesSyncRow{passStartedAt: toNullTime(sync.PassStartedAt), lastCompletedAt: toNullTime(sync.LastCompletedAt)}
	if sync.Cursor != nil {
		row.cursorAccountID = sql.NullString{String: sync.Cursor.AccountID, Valid: true}
		row.cursorDay = sql.NullTime{Time: truncateUTCDay(sync.Cursor.Day), Valid: true}
	}
	return tracing.Trace(span, db.MapSQLError(r.saveSalesSync(ctx, salesRollupSyncName, row)))
}

// salesFactSweepFloor sorts before every invoice; the zero time.Time is below DATETIME's range.
var salesFactSweepFloor = time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)

func truncateUTCDay(t time.Time) time.Time {
	t = t.UTC()
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func truncateUTCMonth(t time.Time) time.Time {
	t = t.UTC()
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
}
