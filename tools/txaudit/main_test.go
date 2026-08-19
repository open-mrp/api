package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The audit is what makes deadlock retry safe, so it has to actually catch each way a callback
// can escape the database — a check that cannot fail would just be reassuring noise.
func TestAudit_CatchesEffectsARollbackWouldNotUndo(t *testing.T) {
	dir := t.TempDir()

	src, err := os.ReadFile(filepath.Join("testdata", "violations.go.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "fixture.go"), src, 0o600); err != nil {
		t.Fatal(err)
	}

	findings, closures, err := audit(dir)
	if err != nil {
		t.Fatal(err)
	}

	if closures != 5 {
		t.Errorf("expected 5 transaction callbacks in the fixture, found %d", closures)
	}

	want := map[string]string{
		"appends to a variable declared outside the callback": "results",
		"calls outside the database":                          "s.stripeClient.Charge",
		"starts a goroutine":                                  "go ...",
		"sends on a channel":                                  "ch <- ...",
	}

	got := map[string]string{}
	for _, f := range findings {
		got[f.kind] = f.detail
	}

	for kind, detail := range want {
		if got[kind] != detail {
			t.Errorf("expected a %q finding for %q, got %q", kind, detail, got[kind])
		}
	}

	// The callback that assembles its result inside and assigns it out once is correct, and
	// flagging it would push people to silence the tool.
	for _, f := range findings {
		if strings.Contains(f.detail, "out") && f.kind == "appends to a variable declared outside the callback" {
			t.Errorf("flagged a callback that builds its result inside the transaction: %s", f.pos)
		}
	}

	if len(findings) != len(want) {
		t.Errorf("expected exactly %d findings, got %d: %+v", len(want), len(findings), findings)
	}
}

// The repository itself must stay clean, since the transaction manager retries on the strength
// of it.
func TestAudit_RepositoryIsClean(t *testing.T) {
	// The repository root, from this package's directory.
	findings, closures, err := audit(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	if closures == 0 {
		t.Fatal("found no transaction callbacks; the scan is not reaching the services")
	}
	for _, f := range findings {
		t.Errorf("%s: %s: %s", f.pos, f.kind, f.detail)
	}
}
