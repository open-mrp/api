package repository

import (
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
