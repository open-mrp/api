package repository

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/open-mrp/api/services/core-service/internal/infrastructure/queries"
)

func TestSubstituteVars_ReplacesKnownVars(t *testing.T) {
	t.Parallel()
	stmt := "INSERT INTO unit (id, account_id) VALUES (@un2, '@account_id')"
	vars := map[string]string{
		"un2": "un_abc123",
	}

	got := substituteVars(stmt, vars)
	want := "INSERT INTO unit (id, account_id) VALUES ('un_abc123', '@account_id')"
	if got != want {
		t.Fatalf("unexpected substitution:\nwant: %s\ngot:  %s", want, got)
	}
}

func TestSubstituteVars_LeavesUnknownVarsAlone(t *testing.T) {
	t.Parallel()
	stmt := "UPDATE department SET location_id = @stloc2 WHERE id = @dept1"
	vars := map[string]string{
		"stloc2": "lc_123",
	}

	got := substituteVars(stmt, vars)
	want := "UPDATE department SET location_id = 'lc_123' WHERE id = @dept1"
	if got != want {
		t.Fatalf("unexpected substitution:\nwant: %s\ngot:  %s", want, got)
	}
}

func TestSubstituteVars_HandlesOverlappingNames(t *testing.T) {
	t.Parallel()
	stmt := "VALUES (@qty10, @qty100)"
	vars := map[string]string{
		"qty10":  "qu_ten",
		"qty100": "qu_hundred",
	}

	got := substituteVars(stmt, vars)
	want := "VALUES ('qu_ten', 'qu_hundred')"
	if got != want {
		t.Fatalf("unexpected substitution:\nwant: %s\ngot:  %s", want, got)
	}
}

func TestParseUserVarSet_MatchesConcatExpr(t *testing.T) {
	t.Parallel()
	stmt := "SET @un2 = CONCAT('un_', LEFT(REPLACE(UUID(), '-', ''), 12))"
	name, expr, ok := parseUserVarSet(stmt)
	if !ok {
		t.Fatal("expected SET @var statement to match")
	}
	if name != "un2" {
		t.Fatalf("unexpected var name: %s", name)
	}
	if expr == "" {
		t.Fatal("expected non-empty expression")
	}
}

func TestParseUserVarSet_MatchesSubquery(t *testing.T) {
	t.Parallel()
	stmt := "SET @un1 = (SELECT id FROM unit WHERE name = 'Each' AND account_id = '@account_id')"
	name, expr, ok := parseUserVarSet(stmt)
	if !ok {
		t.Fatal("expected SET @var statement to match")
	}
	if name != "un1" {
		t.Fatalf("unexpected var name: %s", name)
	}
	if !strings.HasPrefix(expr, "(SELECT") {
		t.Fatalf("expected subquery expression, got: %s", expr)
	}
}

func TestParseUserVarSet_IgnoresSetNames(t *testing.T) {
	t.Parallel()
	stmt := "SET NAMES utf8mb4 COLLATE utf8mb4_unicode_ci"
	_, _, ok := parseUserVarSet(stmt)
	if ok {
		t.Fatal("did not expect SET NAMES to match user-variable parser")
	}
}

// vtgate rejects the FOR UPDATE OF clause outright, and MySQL 8 accepts it.
//
// Production runs PlanetScale; every test in this repository runs against plain mysql:8, including
// the ledger concurrency suite that exists precisely to exercise locking. So this is invisible to
// everything else here: the query passes the prepare smoke test, passes e2e, passes the lock tests,
// and then every allocate_open_issues message dead-letters with
// "Error 1105 (HY000): syntax error at position 191 near 'OF'".
//
// `FOR UPDATE OF a, b` locks exactly the named tables; a bare `FOR UPDATE` locks every table the
// statement reads. They are the same thing whenever the OF list covers the whole FROM clause, which
// is the only shape this codebase had, so dropping the clause cost nothing. If a future query really
// needs to lock a strict subset of its joined tables, it cannot say so on Vitess — split the read
// instead.
func TestVitessCompat_NoForUpdateOfClause(t *testing.T) {
	t.Parallel()

	for _, file := range sqlFiles(t) {
		body, err := os.ReadFile(file) // #nosec G304 -- fixed query directory
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		for _, line := range strings.Split(string(body), "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "--") {
				continue // prose may name the clause to explain why it is banned
			}
			if forUpdateOfRe.MatchString(trimmed) {
				t.Errorf("%s: %q uses FOR UPDATE OF, which vtgate rejects with a 1105 syntax error. "+
					"A bare FOR UPDATE locks every table the statement reads and is what production "+
					"accepts.", filepath.Base(file), trimmed)
			}
		}
	}
}

func TestVitessCompat_NoRawUserVarStatementsReachDB(t *testing.T) {
	t.Parallel()
	stmts := strings.SplitSeq(queries.SandboxSeedSQL, ";")

	for stmt := range stmts {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}

		// Any explicit user-variable SET statement must be intercepted by parseUserVarSet.
		// The seed is split on `;`, so a comment ahead of a SET arrives as part of that
		// statement: the check reads past leading comments rather than only looking at
		// statements that happen to open with SET.
		if strings.HasPrefix(strings.ToUpper(stripLeadingLineComments(stmt)), "SET @") {
			_, _, ok := parseUserVarSet(stmt)
			if !ok {
				t.Fatalf("unrecognized SET @var statement would reach DB: %s", stmt)
			}
		}
	}
}

func TestStripLeadingLineComments_ExposesACommentedSet(t *testing.T) {
	t.Parallel()
	stmt := "-- Invoice line quantities (4)\nSET @qty257 = CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12))"

	name, expr, ok := parseUserVarSet(stmt)
	if !ok {
		t.Fatal("a SET behind a comment must still be recognized as a variable assignment")
	}
	if name != "qty257" {
		t.Fatalf("unexpected var name: %s", name)
	}
	if expr != "CONCAT('qu_', LEFT(REPLACE(UUID(), '-', ''), 12))" {
		t.Fatalf("unexpected expression: %s", expr)
	}
}

func TestStripLeadingLineComments_LeavesOtherStatementsUnmatched(t *testing.T) {
	t.Parallel()
	stmt := "-- SECTION 44\nINSERT INTO production_shift (id) VALUES (@pnsf1)"

	if _, _, ok := parseUserVarSet(stmt); ok {
		t.Fatal("a commented INSERT must not be treated as a variable assignment")
	}
}
