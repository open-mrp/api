//go:build plans

// Package repository's plan tests run list queries against a real MySQL holding a production-shaped
// tenant and fail when a page reads far more of the base table than it returns.
//
// A query plan is invisible from its results: every plan returns the same page, so a wrong one shows
// up only as latency on the one account large enough to feel it. The sqlmock suites never plan, the
// e2e seed is too small for the optimizer to choose the production plan, and asserting index names
// breaks on harmless plan changes. Rows read is the property that matters, so it is what these
// measure — from EXPLAIN ANALYZE, which executes the statement and reports what each access read.
//
// Each request is measured twice: under statistics freshly analyzed from the corpus, and under a
// snapshot of production's (testdata/plan_stats). Production's are sampled from a handful of pages
// and can be far from the truth; the plan that read 1.4M rows for one transaction page in production
// came from them, and the local optimizer only reproduces it under the same numbers.
//
// `make test-plans` runs them against the local dev MySQL; the corpus they seed is kept between runs.
package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
)

// interpolateParams matches shared/db/db_pool.go, so statements reach MySQL the way production sends them.
const defaultPlanDSN = "root:Testing123!@tcp(127.0.0.1:3306)/openmrp?parseTime=true&loc=UTC&interpolateParams=true"

var (
	planDBOnce sync.Once
	planSQLDB  *sql.DB
	planDBErr  error
)

func planDB(t *testing.T) *sql.DB {
	t.Helper()
	planDBOnce.Do(func() {
		dsn := os.Getenv("PLAN_TEST_DSN")
		if dsn == "" {
			dsn = defaultPlanDSN
		}
		planSQLDB, planDBErr = sql.Open("mysql", dsn)
		if planDBErr == nil {
			planDBErr = planSQLDB.Ping()
		}
	})
	require.NoError(t, planDBErr, "connecting to the plan test database (make local-db, or set PLAN_TEST_DSN)")
	return planSQLDB
}

// explainingDB is a sqlc.DBTX that runs EXPLAIN ANALYZE ahead of every SELECT and keeps each
// statement with its plan.
type explainingDB struct {
	db         *sql.DB
	statements []explainedStatement
}

type explainedStatement struct {
	query string
	args  []any
	plan  string
}

func (e *explainingDB) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	if strings.HasPrefix(stripLeadingComments(query), "SELECT") {
		plan, err := explainAnalyze(ctx, e.db, query, args...)
		if err != nil {
			return nil, err
		}
		e.statements = append(e.statements, explainedStatement{query: query, args: args, plan: plan})
	}
	return e.db.QueryContext(ctx, query, args...)
}

// stripLeadingComments drops the "-- name: ..." lines sqlc puts ahead of its statements.
func stripLeadingComments(query string) string {
	query = strings.TrimSpace(query)
	for strings.HasPrefix(query, "--") {
		_, rest, _ := strings.Cut(query, "\n")
		query = strings.TrimSpace(rest)
	}
	return query
}

func explainAnalyze(ctx context.Context, db *sql.DB, query string, args ...any) (string, error) {
	var plan string
	err := db.QueryRowContext(ctx, "EXPLAIN ANALYZE "+query, args...).Scan(&plan)
	return plan, err
}

// scopeIndexes lists table's B-tree indexes that lead with scopeColumn (account_id, owner_account_id):
// the ones a tenant-scoped list can be driven from.
func scopeIndexes(t *testing.T, db *sql.DB, table, scopeColumn string) []string {
	t.Helper()
	rows, err := db.Query(`SELECT DISTINCT index_name FROM information_schema.statistics
		WHERE table_schema = DATABASE() AND table_name = ? AND seq_in_index = 1
		  AND column_name = ? AND index_type = 'BTREE'`, table, scopeColumn)
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()
	var out []string
	for rows.Next() {
		var name string
		require.NoError(t, rows.Scan(&name))
		out = append(out, name)
	}
	require.NoError(t, rows.Err())
	require.NotEmpty(t, out)
	return out
}

// bestForcedAccess replays stmt with each index forced on alias and returns the cheapest read, the
// statement's own plan included, so a test can tell a planner that chose badly from a query no index
// can serve. from is the statement's FROM item for alias, e.g. "FROM `transaction` t"; any hint
// already on it is replaced.
func bestForcedAccess(t *testing.T, db *sql.DB, stmt explainedStatement, from, alias string, indexes []string) (planAccess, string) {
	t.Helper()
	hinted := regexp.MustCompile(regexp.QuoteMeta(from) + `(?: FORCE INDEX \([^)]*\))?`)
	require.True(t, hinted.MatchString(stmt.query), "statement has no %q", from)

	best := tableAccess(stmt.plan, alias)
	bestIndex := "its own plan"
	for _, index := range indexes {
		query := hinted.ReplaceAllLiteralString(stmt.query, from+" FORCE INDEX ("+index+")")
		plan, err := explainAnalyze(context.Background(), db, query, stmt.args...)
		if err != nil {
			continue // e.g. a MATCH that needs its FULLTEXT index
		}
		if a := tableAccess(plan, alias); a.rows < best.rows {
			best, bestIndex = a, index
		}
	}
	return best, bestIndex
}

func (e *explainingDB) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return e.db.QueryRowContext(ctx, query, args...)
}

func (e *explainingDB) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return e.db.ExecContext(ctx, query, args...)
}

func (e *explainingDB) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	return e.db.PrepareContext(ctx, query)
}

// planAccessRe matches an access node on one table alias in EXPLAIN ANALYZE's tree, e.g.
// "-> Index range scan on t using idx over (...)  (cost=.. rows=..) (actual time=0.1..2.3 rows=26 loops=1)".
var planAccessRe = regexp.MustCompile(`-> (.*?) on (\S+)(?: using (\S+))?.*\(actual time=\S+ rows=(\S+) loops=(\d+)\)`)

// planAccess is what a plan read from one table alias.
type planAccess struct {
	rows    float64
	indexes []string
}

// tableAccess sums the rows every access node on alias read, across all its loops.
func tableAccess(plan, alias string) planAccess {
	var a planAccess
	for _, line := range strings.Split(plan, "\n") {
		m := planAccessRe.FindStringSubmatch(line)
		if m == nil || m[2] != alias {
			continue
		}
		rows, _ := strconv.ParseFloat(m[4], 64)
		loops, _ := strconv.ParseFloat(m[5], 64)
		a.rows += rows * loops
		if m[3] != "" {
			a.indexes = append(a.indexes, m[3])
		} else {
			a.indexes = append(a.indexes, m[1])
		}
	}
	return a
}

var planRootRe = regexp.MustCompile(`\(actual time=\S+ rows=(\S+) loops=1\)`)

// planReturned is how many rows the statement returned: the actual rows of the plan's root node.
func planReturned(plan string) float64 {
	root, _, _ := strings.Cut(plan, "\n")
	m := planRootRe.FindStringSubmatch(root)
	if m == nil {
		return 0
	}
	rows, _ := strconv.ParseFloat(m[1], 64)
	return rows
}

// planStats is a snapshot of one table's InnoDB persistent statistics. Refresh one by running, on the
// production branch,
//
//	SELECT n_rows, clustered_index_size, sum_of_other_index_sizes FROM mysql.innodb_table_stats
//	  WHERE database_name = DATABASE() AND table_name = '<table>';
//	SELECT index_name, stat_name, stat_value FROM mysql.innodb_index_stats
//	  WHERE database_name = DATABASE() AND table_name = '<table>';
//
// and writing the rows to testdata/plan_stats/<table>.json.
type planStats struct {
	Source string `json:"source"`
	Table  struct {
		NRows                int64 `json:"n_rows"`
		ClusteredIndexSize   int64 `json:"clustered_index_size"`
		SumOfOtherIndexSizes int64 `json:"sum_of_other_index_sizes"`
	} `json:"table"`
	Indexes []struct {
		Index string `json:"index"`
		Stat  string `json:"stat"`
		Value int64  `json:"value"`
	} `json:"indexes"`
}

// planStatsModes are the statistics every plan test runs under.
var planStatsModes = []string{"analyzed", "production"}

// usePlanStats loads table's statistics for mode: "analyzed" recomputes them from the rows, and
// "production" overwrites them with the snapshot. An index the snapshot does not name (one added
// since) keeps its analyzed numbers.
func usePlanStats(t *testing.T, db *sql.DB, table, mode string) {
	t.Helper()
	_, err := db.Exec("ANALYZE TABLE `" + table + "`")
	require.NoError(t, err)
	if mode == "analyzed" {
		return
	}
	require.Equal(t, "production", mode)

	raw, err := os.ReadFile(filepath.Join("testdata", "plan_stats", table+".json"))
	require.NoError(t, err)
	var stats planStats
	require.NoError(t, json.Unmarshal(raw, &stats))

	_, err = db.Exec(`UPDATE mysql.innodb_table_stats SET n_rows = ?, clustered_index_size = ?, sum_of_other_index_sizes = ?
		WHERE database_name = DATABASE() AND table_name = ?`,
		stats.Table.NRows, stats.Table.ClusteredIndexSize, stats.Table.SumOfOtherIndexSizes, table)
	require.NoError(t, err)
	for _, s := range stats.Indexes {
		_, err = db.Exec(`UPDATE mysql.innodb_index_stats SET stat_value = ?
			WHERE database_name = DATABASE() AND table_name = ? AND index_name = ? AND stat_name = ?`,
			s.Value, table, s.Index, s.Stat)
		require.NoError(t, err)
	}
	// The server reads persistent statistics when it opens the table.
	_, err = db.Exec("FLUSH TABLE `" + table + "`")
	require.NoError(t, err)
}

// planValue is one setting of one list filter.
type planValue[P any] struct {
	label string
	apply func(*P)
}

// planDim is one list filter and the values a suite tries for it: a common one and a rare one at least.
type planDim[P any] struct {
	name   string
	values []planValue[P]
}

// planCase is one list request: a filter combination on one page.
type planCase[P any] struct {
	name   string
	params P
}

// planCases is every filter value alone and every pair of values across two filters, each on every
// page. Pairs are enough: the guarantee a list owes is that some index pins its most selective filter
// and every other filter is residual. A page applies a cursor; give at least the first page and one
// deep in the tenant in each direction.
func planCases[P any](base P, dims []planDim[P], pages []planValue[P]) []planCase[P] {
	var combos [][]planValue[P]
	combos = append(combos, nil)
	for i, d := range dims {
		for _, v := range d.values {
			combos = append(combos, []planValue[P]{v})
			for _, d2 := range dims[i+1:] {
				for _, v2 := range d2.values {
					combos = append(combos, []planValue[P]{v, v2})
				}
			}
		}
	}
	var cases []planCase[P]
	for _, combo := range combos {
		labels := []string{}
		for _, v := range combo {
			labels = append(labels, v.label)
		}
		if len(labels) == 0 {
			labels = []string{"unfiltered"}
		}
		for _, page := range pages {
			p := base
			page.apply(&p)
			for _, v := range combo {
				v.apply(&p)
			}
			cases = append(cases, planCase[P]{name: strings.Join(labels, ",") + "/" + page.label, params: p})
		}
	}
	return cases
}

// listPlanSuite is one list endpoint's plan test: every case, run against a seeded corpus under both
// statistics modes, and held to reading about a page of its table.
type listPlanSuite[P any] struct {
	// table is the listed table: its statistics are swapped, and its scope-led indexes are the
	// candidates each statement is replayed under.
	table, scopeColumn string
	// from is the FROM item the page is read through ("FROM `transaction` t"), whose hint the replays
	// replace, and alias is its alias. Of the statements containing from, the one that reads the most of
	// the table is measured (a list may choose a page in one and hydrate it in another).
	from, alias string
	cases       []planCase[P]
	limit       func(P) int32
	// list runs the request against q, whose statements the suite explains.
	list func(ctx context.Context, q *sqlc.Queries, p P) error
	// floor (optional) is how many rows a request's unordered filter matches, or 0 when it has none: a
	// filter no index can serve in list order (a FULLTEXT match, a range on a column the list does not
	// sort by) cannot stop at a page, so the bar for such a request is reading only its matches.
	floor func(t *testing.T, db *sql.DB, p P) float64
}

// planRowBudget is the most of the listed table one page may read: the page, plus room for residual
// filters that reject some rows the driving index yields.
func planRowBudget(limit int32) float64 { return float64(10 * (limit + 1)) }

// run checks every case under both statistics modes. Each request is replayed with every scope-led
// index forced, and two things are asserted separately:
//   - the plan it got reads about what the best of those indexes reads: a miss is the planner (or a
//     hint) choosing badly, fixed in the query;
//   - the best index reads about a page: a miss is an index that does not exist, fixed in a migration.
//
// A request that returns less than it asked for is exempt from the second: whichever index drives it,
// finding there is no next row means reading that index's range to its end, which no index avoids.
// One with an unordered filter is held to that filter's matches instead (floor).
func (s listPlanSuite[P]) run(t *testing.T) {
	db := planDB(t)
	edb := &explainingDB{db: db}
	q := sqlc.New(edb)
	indexes := scopeIndexes(t, db, s.table, s.scopeColumn)

	read := 0
	for _, mode := range planStatsModes {
		t.Run("stats="+mode, func(t *testing.T) {
			usePlanStats(t, db, s.table, mode)
			t.Cleanup(func() { usePlanStats(t, db, s.table, "analyzed") })
			for _, tc := range s.cases {
				t.Run(tc.name, func(t *testing.T) {
					if s.check(t, db, edb, q, indexes, tc) {
						read++
					}
				})
			}
		})
	}
	// A request answered without a query (a filter that resolved to nothing) reads nothing, but a suite
	// in which none read the table is measuring the wrong statement.
	require.Positive(t, read, "no request read %q", s.from)
}

// check measures one request and reports whether it read the table at all.
func (s listPlanSuite[P]) check(t *testing.T, db *sql.DB, edb *explainingDB, q *sqlc.Queries, indexes []string, tc planCase[P]) bool {
	t.Helper()

	edb.statements = nil
	require.NoError(t, s.list(context.Background(), q, tc.params))
	var stmt *explainedStatement
	for i := range edb.statements {
		if strings.Contains(edb.statements[i].query, s.from) &&
			(stmt == nil || tableAccess(edb.statements[i].plan, s.alias).rows > tableAccess(stmt.plan, s.alias).rows) {
			stmt = &edb.statements[i]
		}
	}
	if stmt == nil {
		return false
	}

	got := tableAccess(stmt.plan, s.alias)
	limit := s.limit(tc.params)
	page := float64(limit + 1)
	budget := planRowBudget(limit)
	if got.rows <= budget {
		return true
	}
	// Reading no more than the request's unordered filter matches (and the page's rows again, if they
	// are joined back by id) is as well as any plan can do; within twice that is the same allowance a
	// plan gets against the best forced index, which lets it read another filter's range whole instead.
	var floor float64
	if s.floor != nil {
		floor = s.floor(t, db, tc.params)
	}
	if floor > 0 && got.rows <= 2*floor+2*page {
		return true
	}
	best, bestIndex := bestForcedAccess(t, db, *stmt, s.from, s.alias, indexes)

	switch {
	case got.rows > 2*best.rows+page:
		t.Errorf("read %.0f %s rows via %v; forcing %s reads %.0f\n%s", got.rows, s.table, got.indexes, bestIndex, best.rows, stmt.plan)
	case floor > 0:
		t.Errorf("read %.0f %s rows via %v; its unordered filter matches only %.0f\n%s", got.rows, s.table, got.indexes, floor, stmt.plan)
	case best.rows > budget && planReturned(stmt.plan) >= page:
		t.Errorf("no index serves this: the best, %s, reads %.0f %s rows to return a page of %d\n%s",
			bestIndex, best.rows, s.table, limit, stmt.plan)
	}
	return true
}
