package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The whole point of this tool is to fail on a statement MySQL accepts, so the test drives it with the real one: `FOR UPDATE OF ia, q` shipped green through every check in this repository and then dead-lettered every allocate_open_issues message in production.
//
// The Postgres fixture is the other half. Its $1 placeholders are a parse error to this parser, so a tool that scanned every service rather than reading each sqlc.yaml would report agent-service as broken forever and be turned off within a day.
func TestRun_RejectsWhatVtgateRejectsAndSkipsPostgres(t *testing.T) {
	root := t.TempDir()
	writeService(t, root, "mysql-svc", "mysql", "mysql_queries.go.txt")
	writeService(t, root, "pg-svc", "postgresql", "postgres_queries.go.txt")

	findings, total, err := run(root)
	if err != nil {
		t.Fatal(err)
	}

	// Four statements in the MySQL fixture; the bare string const is not one, and the Postgres service is not read at all.
	if total != 4 {
		t.Errorf("expected 4 generated queries examined, got %d", total)
	}

	if len(findings) != 1 {
		t.Fatalf("expected exactly 1 finding, got %d: %+v", len(findings), findings)
	}
	if findings[0].name != "readIssueCoverageForUpdate" {
		t.Errorf("expected the FOR UPDATE OF query to be named, got %q", findings[0].name)
	}
	if !strings.Contains(findings[0].err.Error(), "OF") {
		t.Errorf("expected the parse error to point at the OF clause, got %v", findings[0].err)
	}
	if !strings.HasSuffix(findings[0].pos.Filename, "queries.sql.go") {
		t.Errorf("expected the finding to carry its file, got %q", findings[0].pos.Filename)
	}
	if findings[0].pos.Line == 0 {
		t.Error("expected the finding to carry a line number, so the failure names the query to fix")
	}
}

// A root with no MySQL service is a misconfigured invocation, not a clean run: reporting "all queries parse" over zero queries is how this check silently stops running.
func TestRun_FailsWhenNothingWasScanned(t *testing.T) {
	if _, _, err := run(t.TempDir()); err == nil {
		t.Fatal("expected an error when no MySQL sqlc config was found")
	}
}

func writeService(t *testing.T, root, name, engine, fixture string) {
	t.Helper()

	out := filepath.Join(root, "services", name, "internal", "infrastructure", "sqlc")
	if err := os.MkdirAll(out, 0o750); err != nil {
		t.Fatal(err)
	}

	config := "version: \"2\"\nsql:\n  - queries: \"internal/infrastructure/queries\"\n    engine: \"" + engine +
		"\"\n    gen:\n      go:\n        out: \"internal/infrastructure/sqlc\"\n"
	if err := os.WriteFile(filepath.Join(root, "services", name, "sqlc.yaml"), []byte(config), 0o600); err != nil {
		t.Fatal(err)
	}

	src, err := os.ReadFile(filepath.Join("testdata", fixture))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(out, "queries.sql.go"), src, 0o600); err != nil {
		t.Fatal(err)
	}
}
