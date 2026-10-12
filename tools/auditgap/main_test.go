package main

import (
	"path/filepath"
	"testing"
)

func TestCheck_FlagsHandlersThatWriteWithoutAuditing(t *testing.T) {
	dir, err := filepath.Abs(filepath.Join("testdata", "fixture"))
	if err != nil {
		t.Fatal(err)
	}
	allow := allowlist{
		"svc/handler.Exempt": "fixture exemption",
		"svc/handler.Gone":   "no longer exists",
	}

	gaps, stale, handlers, err := check(dir, []string{"./..."}, allow, "")
	if err != nil {
		t.Fatal(err)
	}

	if handlers != 6 {
		t.Errorf("expected the 6 served methods as roots, got %d", handlers)
	}
	got := map[string]string{}
	for _, g := range gaps {
		got[g.handler] = g.writes[0]
	}
	want := map[string]string{"svc/handler.Unaudited": "CreateThing", "svc/handler.Indirect": "CreateThing"}
	if len(got) != len(want) {
		t.Errorf("gaps = %v, want %v", got, want)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("gap %s = %q, want %q (all gaps: %v)", k, got[k], v, got)
		}
	}
	if len(stale) != 1 || stale[0] != "svc/handler.Gone" {
		t.Errorf("stale = %v, want the entry that matches no handler", stale)
	}
}
