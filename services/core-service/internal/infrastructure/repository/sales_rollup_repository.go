package repository

import (
	"context"
	"crypto/md5" // #nosec G501 - a key, not a security boundary
	"database/sql"
	"errors"
	"fmt"
	"math/big"
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

// salesRollupDimension is how one dimension's bucket rows are derived from a fact.
type salesRollupDimension struct {
	name string
	// id is the fact's group in the dimension, ok false when it has none; nil for the account-wide total.
	id func(f *rollupFact) (id string, ok bool)
	// perProductLine adds a row per product line beside the all-product-line row. The product line dimension needs none: its own key already is the product line.
	perProductLine bool
	// hourly also keeps hour buckets, which the daily chart regroups into local days.
	hourly bool
}

var salesRollupDimensions = []salesRollupDimension{
	{name: rollupDimTotal, perProductLine: true, hourly: true},
	{name: rollupDimBuyer, id: func(f *rollupFact) (string, bool) { return f.buyerAccountID, true }, perProductLine: true},
	{name: rollupDimItem, id: func(f *rollupFact) (string, bool) { return f.itemID, true }, perProductLine: true},
	{name: rollupDimProductLine, id: func(f *rollupFact) (string, bool) { return f.productLineID, true }},
	// Lines without a sales rep or discount belong to no group of those dimensions.
	{name: rollupDimSalesRep, id: func(f *rollupFact) (string, bool) { return f.salesRepID.String, f.salesRepID.Valid }, perProductLine: true},
	{name: rollupDimDiscount, id: func(f *rollupFact) (string, bool) { return f.orderDiscountID.String, f.orderDiscountID.Valid }, perProductLine: true},
}

const rollupColumns = `account_id, sales_order_type_code, dimension, product_line_key, grain, bucket_start, row_hash, dimension_id, sales_rep_key,
quantity_base, total_invoiced, total_cost, invoice_count, line_count`

// rollupWriteBatch bounds the rows one rollup statement writes or deletes, keeping each under 25ms.
const rollupWriteBatch = 50

// rollupRowHash is MD5(CONCAT(dimension_id, CHAR(0), sales_rep_key)), the key's stand-in for the pair; NUL occurs in neither.
func rollupRowHash(dimensionID, salesRepKey string) string {
	sum := md5.Sum([]byte(dimensionID + "\x00" + salesRepKey)) // #nosec G401 - a key, not a security boundary
	return string(sum[:])
}

// rollupKey is a sales_fact_rollup primary key within one account.
type rollupKey struct {
	typeCode, dimension, lineKey, grain string
	bucket                              time.Time
	hash                                string
}

// rollupRow is one sales_fact_rollup row within one account. Amounts are DECIMAL(28,10) text, or invalid for NULL.
type rollupRow struct {
	key                     rollupKey
	dimensionID, salesRep   string
	qty, invoiced, cost     sql.NullString
	invoiceCount, lineCount int64
}

func (a rollupRow) equal(b rollupRow) bool {
	return a.key == b.key && a.dimensionID == b.dimensionID && a.salesRep == b.salesRep &&
		a.qty == b.qty && a.invoiced == b.invoiced && a.cost == b.cost &&
		a.invoiceCount == b.invoiceCount && a.lineCount == b.lineCount
}

// rollupFact is the part of a sales_line_fact the rollups sum.
type rollupFact struct {
	invoicedAt                                  time.Time
	invoiceID, typeCode, buyerAccountID, itemID string
	productLineID                               string
	salesRepID, orderDiscountID                 sql.NullString
	qty, invoiced, cost                         sql.NullString
}

// rollupSum accumulates one rollup row the way SUM and COUNT(DISTINCT) do: a sum of only NULLs is NULL.
type rollupSum struct {
	row                 rollupRow
	qty, invoiced, cost *big.Rat
	invoices            map[string]struct{}
}

func addDecimal(sum **big.Rat, v sql.NullString) error {
	if !v.Valid {
		return nil
	}
	r, ok := new(big.Rat).SetString(v.String)
	if !ok {
		return fmt.Errorf("unreadable decimal %q", v.String)
	}
	if *sum == nil {
		*sum = new(big.Rat)
	}
	(*sum).Add(*sum, r)
	return nil
}

// rollupDecimal formats a sum as DECIMAL(28,10) stores it: rounded half away from zero, as MySQL rounds on insert.
func rollupDecimal(r *big.Rat) sql.NullString {
	if r == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: r.FloatString(10), Valid: true}
}

// canonicalDecimal rewrites stored DECIMAL text in rollupDecimal's form, so equal amounts compare equal as strings.
func canonicalDecimal(v sql.NullString) (sql.NullString, error) {
	if !v.Valid {
		return v, nil
	}
	r, ok := new(big.Rat).SetString(v.String)
	if !ok {
		return v, fmt.Errorf("unreadable decimal %q", v.String)
	}
	return rollupDecimal(r), nil
}

type rollupBucket struct {
	grain string
	start time.Time
}

// rollupDayRows sums one day's facts into every hour and day bucket row the rollups keep for it.
func rollupDayRows(day time.Time, facts []rollupFact) (map[rollupKey]rollupRow, error) {
	sums := map[rollupKey]*rollupSum{}
	for i := range facts {
		f := &facts[i]
		for _, d := range salesRollupDimensions {
			dimID := ""
			if d.id != nil {
				id, ok := d.id(f)
				if !ok {
					continue
				}
				dimID = id
			}
			rep := f.salesRepID.String
			hash := rollupRowHash(dimID, rep)
			lineKeys := []string{""}
			if d.perProductLine {
				lineKeys = append(lineKeys, f.productLineID)
			}
			buckets := []rollupBucket{{rollupGrainDay, day}}
			if d.hourly {
				buckets = append(buckets, rollupBucket{rollupGrainHour, f.invoicedAt.UTC().Truncate(time.Hour)})
			}
			for _, lineKey := range lineKeys {
				for _, b := range buckets {
					k := rollupKey{typeCode: f.typeCode, dimension: d.name, lineKey: lineKey, grain: b.grain, bucket: b.start, hash: hash}
					s, ok := sums[k]
					if !ok {
						s = &rollupSum{row: rollupRow{key: k, dimensionID: dimID, salesRep: rep}, invoices: map[string]struct{}{}}
						sums[k] = s
					}
					if err := addDecimal(&s.qty, f.qty); err != nil {
						return nil, err
					}
					if err := addDecimal(&s.invoiced, f.invoiced); err != nil {
						return nil, err
					}
					if err := addDecimal(&s.cost, f.cost); err != nil {
						return nil, err
					}
					s.invoices[f.invoiceID] = struct{}{}
					s.row.lineCount++
				}
			}
		}
	}
	out := make(map[rollupKey]rollupRow, len(sums))
	for k, s := range sums {
		row := s.row
		row.qty, row.invoiced, row.cost = rollupDecimal(s.qty), rollupDecimal(s.invoiced), rollupDecimal(s.cost)
		row.invoiceCount = int64(len(s.invoices))
		out[k] = row
	}
	return out, nil
}

// diffRollupRows returns the wanted rows that are missing or differ from what is stored, and the stored keys no longer wanted.
func diffRollupRows(want, stored map[rollupKey]rollupRow) (upserts []rollupRow, deletes []rollupKey) {
	for k, w := range want {
		if s, ok := stored[k]; !ok || !s.equal(w) {
			upserts = append(upserts, w)
		}
	}
	for k := range stored {
		if _, ok := want[k]; !ok {
			deletes = append(deletes, k)
		}
	}
	return upserts, deletes
}

// RebuildRollupDay brings a day's hour and day buckets in line with its facts, writing only rows that differ, then recomputes the month rows those changes reach.
func (r *salesFactRepoImpl) RebuildRollupDay(ctx context.Context, day domain.SalesRollupDay) *apierror.APIError {
	ctx, span := salesFactRepoTracer.Start(ctx, "repository.sales_fact.rebuild_rollup_day")
	defer span.End()

	start := truncateUTCDay(day.Day)
	end := start.AddDate(0, 0, 1)
	return tracing.Trace(span, r.inTx(ctx, func(tx sqlc.DBTX) error {
		facts, err := selectRollupFacts(ctx, tx, day.AccountID, start, end)
		if err != nil {
			return err
		}
		want, err := rollupDayRows(start, facts)
		if err != nil {
			return err
		}
		stored, err := selectRollupRows(ctx, tx, `SELECT `+rollupRowColumns+` FROM sales_fact_rollup
WHERE account_id = ? AND bucket_start >= ? AND bucket_start < ? AND grain IN ('hour', 'day')`, day.AccountID, start, end)
		if err != nil {
			return err
		}
		upserts, deletes := diffRollupRows(want, stored)
		if len(upserts) == 0 && len(deletes) == 0 {
			return nil
		}
		if err := upsertRollupRows(ctx, tx, day.AccountID, upserts); err != nil {
			return err
		}
		if err := deleteRollupRows(ctx, tx, day.AccountID, deletes); err != nil {
			return err
		}
		return refreshRollupMonths(ctx, tx, day.AccountID, truncateUTCMonth(start), touchedDayRows(upserts, deletes, stored))
	}))
}

// touchedDayRows returns the day-grain rows a rebuild wrote or deleted; only their months can have changed.
func touchedDayRows(upserts []rollupRow, deletes []rollupKey, stored map[rollupKey]rollupRow) []rollupRow {
	var out []rollupRow
	for _, row := range upserts {
		if row.key.grain == rollupGrainDay {
			out = append(out, row)
		}
	}
	for _, k := range deletes {
		if k.grain == rollupGrainDay {
			out = append(out, stored[k])
		}
	}
	return out
}

// refreshRollupMonths recomputes, from the day rows, each month row that shares a key with a touched day row.
func refreshRollupMonths(ctx context.Context, tx sqlc.DBTX, accountID string, month time.Time, touched []rollupRow) error {
	type slice struct{ dimension, lineKey string }
	monthKey := func(k rollupKey) rollupKey {
		k.grain, k.bucket = rollupGrainMonth, month
		return k
	}
	wanted := map[rollupKey]struct{}{}
	idsBySlice := map[slice][]string{}
	seenID := map[slice]map[string]struct{}{}
	for _, row := range touched {
		wanted[monthKey(row.key)] = struct{}{}
		s := slice{row.key.dimension, row.key.lineKey}
		if seenID[s] == nil {
			seenID[s] = map[string]struct{}{}
		}
		if _, ok := seenID[s][row.dimensionID]; !ok {
			seenID[s][row.dimensionID] = struct{}{}
			idsBySlice[s] = append(idsBySlice[s], row.dimensionID)
		}
	}

	found := map[rollupKey]rollupRow{}
	for s, ids := range idsBySlice {
		for i := 0; i < len(ids); i += rollupWriteBatch {
			batch := ids[i:min(i+rollupWriteBatch, len(ids))]
			args := append([]any{accountID, s.dimension, s.lineKey}, stringsToAny(batch)...)
			rows, err := selectRollupRows(ctx, tx, `SELECT sales_order_type_code, dimension, product_line_key, 'month', MIN(bucket_start), row_hash, dimension_id, sales_rep_key,
SUM(quantity_base), SUM(total_invoiced), SUM(total_cost), SUM(invoice_count), SUM(line_count)
FROM sales_fact_rollup FORCE INDEX (sales_fact_rollup_dimension_id_idx)
WHERE account_id = ? AND dimension = ? AND product_line_key = ? AND dimension_id IN (`+placeholders(len(batch))+`)
  AND grain = 'day' AND bucket_start >= ? AND bucket_start < ?
GROUP BY sales_order_type_code, dimension, product_line_key, row_hash, dimension_id, sales_rep_key`, append(args, month, month.AddDate(0, 1, 0))...)
			if err != nil {
				return err
			}
			for _, row := range rows {
				row.key = monthKey(row.key)
				if _, ok := wanted[row.key]; ok {
					found[row.key] = row
				}
			}
		}
	}

	upserts := make([]rollupRow, 0, len(found))
	for _, row := range found {
		upserts = append(upserts, row)
	}
	var deletes []rollupKey
	for k := range wanted {
		if _, ok := found[k]; !ok {
			deletes = append(deletes, k)
		}
	}
	if err := upsertRollupRows(ctx, tx, accountID, upserts); err != nil {
		return err
	}
	return deleteRollupRows(ctx, tx, accountID, deletes)
}

func selectRollupFacts(ctx context.Context, tx sqlc.DBTX, accountID string, start, end time.Time) ([]rollupFact, error) {
	rows, err := tx.QueryContext(ctx, `SELECT invoiced_at, invoice_id, sales_order_type_code, buyer_account_id, sales_rep_id, order_discount_id,
item_id, product_line_id, quantity_base, total_invoiced, total_cost
FROM sales_line_fact WHERE account_id = ? AND invoiced_at >= ? AND invoiced_at < ?`, accountID, start, end)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []rollupFact
	for rows.Next() {
		var f rollupFact
		if err := rows.Scan(&f.invoicedAt, &f.invoiceID, &f.typeCode, &f.buyerAccountID, &f.salesRepID, &f.orderDiscountID,
			&f.itemID, &f.productLineID, &f.qty, &f.invoiced, &f.cost); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

// rollupRowColumns are the columns selectRollupRows scans, account_id aside.
const rollupRowColumns = `sales_order_type_code, dimension, product_line_key, grain, bucket_start, row_hash, dimension_id, sales_rep_key,
quantity_base, total_invoiced, total_cost, invoice_count, line_count`

func selectRollupRows(ctx context.Context, tx sqlc.DBTX, query string, args ...any) (map[rollupKey]rollupRow, error) {
	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := map[rollupKey]rollupRow{}
	for rows.Next() {
		var (
			row  rollupRow
			hash []byte
		)
		if err := rows.Scan(&row.key.typeCode, &row.key.dimension, &row.key.lineKey, &row.key.grain, &row.key.bucket, &hash,
			&row.dimensionID, &row.salesRep, &row.qty, &row.invoiced, &row.cost, &row.invoiceCount, &row.lineCount); err != nil {
			return nil, err
		}
		row.key.bucket = row.key.bucket.UTC()
		row.key.hash = string(hash)
		for _, v := range []*sql.NullString{&row.qty, &row.invoiced, &row.cost} {
			if *v, err = canonicalDecimal(*v); err != nil {
				return nil, err
			}
		}
		out[row.key] = row
	}
	return out, rows.Err()
}

func upsertRollupRows(ctx context.Context, tx sqlc.DBTX, accountID string, rows []rollupRow) error {
	for i := 0; i < len(rows); i += rollupWriteBatch {
		batch := rows[i:min(i+rollupWriteBatch, len(rows))]
		args := make([]any, 0, 14*len(batch))
		for _, row := range batch {
			k := row.key
			args = append(args, accountID, k.typeCode, k.dimension, k.lineKey, k.grain, k.bucket, []byte(k.hash), row.dimensionID, row.salesRep,
				row.qty, row.invoiced, row.cost, row.invoiceCount, row.lineCount)
		}
		values := strings.TrimSuffix(strings.Repeat("(?,?,?,?,?,?,?,?,?,?,?,?,?,?),", len(batch)), ",")
		if _, err := tx.ExecContext(ctx, `INSERT INTO sales_fact_rollup (`+rollupColumns+`) VALUES `+values+`
ON DUPLICATE KEY UPDATE dimension_id = VALUES(dimension_id), sales_rep_key = VALUES(sales_rep_key), quantity_base = VALUES(quantity_base),
total_invoiced = VALUES(total_invoiced), total_cost = VALUES(total_cost), invoice_count = VALUES(invoice_count), line_count = VALUES(line_count)`, args...); err != nil {
			return err
		}
	}
	return nil
}

func deleteRollupRows(ctx context.Context, tx sqlc.DBTX, accountID string, keys []rollupKey) error {
	for i := 0; i < len(keys); i += rollupWriteBatch {
		batch := keys[i:min(i+rollupWriteBatch, len(keys))]
		args := make([]any, 0, 1+6*len(batch))
		args = append(args, accountID)
		for _, k := range batch {
			args = append(args, k.typeCode, k.dimension, k.lineKey, k.grain, k.bucket, []byte(k.hash))
		}
		tuples := strings.TrimSuffix(strings.Repeat("(?,?,?,?,?,?),", len(batch)), ",")
		if _, err := tx.ExecContext(ctx, `DELETE FROM sales_fact_rollup WHERE account_id = ?
AND (sales_order_type_code, dimension, product_line_key, grain, bucket_start, row_hash) IN (`+tuples+`)`, args...); err != nil {
			return err
		}
	}
	return nil
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
