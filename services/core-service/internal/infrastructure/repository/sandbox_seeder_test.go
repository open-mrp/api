package repository

import (
	"strings"
	"testing"

	"github.com/augno/api/services/core-service/internal/infrastructure/queries"
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
		if strings.HasPrefix(strings.ToUpper(stmt), "SET @") {
			_, _, ok := parseUserVarSet(stmt)
			if !ok {
				t.Fatalf("unrecognized SET @var statement would reach DB: %s", stmt)
			}
		}
	}
}
