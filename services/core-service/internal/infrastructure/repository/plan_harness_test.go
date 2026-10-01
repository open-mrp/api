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
	if strings.HasPrefix(strings.TrimSpace(query), "SELECT") {
		plan, err := explainAnalyze(ctx, e.db, query, args...)
		if err != nil {
			return nil, err
		}
		e.statements = append(e.statements, explainedStatement{query: query, args: args, plan: plan})
	}
	return e.db.QueryContext(ctx, query, args...)
}

func explainAnalyze(ctx context.Context, db *sql.DB, query string, args ...any) (string, error) {
	var plan string
	err := db.QueryRowContext(ctx, "EXPLAIN ANALYZE "+query, args...).Scan(&plan)
	return plan, err
}

// accountIndexes lists table's B-tree indexes that lead with account_id: the ones a tenant-scoped list
// can be driven from.
func accountIndexes(t *testing.T, db *sql.DB, table string) []string {
	t.Helper()
	rows, err := db.Query(`SELECT DISTINCT index_name FROM information_schema.statistics
		WHERE table_schema = DATABASE() AND table_name = ? AND seq_in_index = 1
		  AND column_name = 'account_id' AND index_type = 'BTREE'`, table)
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
