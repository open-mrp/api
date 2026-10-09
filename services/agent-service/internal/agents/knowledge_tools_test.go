package agents

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/open-mrp/api/services/agent-service/internal/domain"
)

const sampleLLMsIndex = `# Docs

## Get Started
- [Overview](https://docs.example.com/get-started/overview.md): What the platform does.

## API Reference

### Sales
- [Create a sales order](https://docs.example.com/api-reference/sales/create-sales-order.md): Creates a sales order.
- [List customers](https://docs.example.com/api-reference/sales/list-customers.md): Returns a paginated list of customers.
`

func TestParseLLMsIndex(t *testing.T) {
	t.Parallel()
	entries := parseLLMsIndex(sampleLLMsIndex)
	if len(entries) != 3 {
		t.Fatalf("want 3 entries, got %+v", entries)
	}
	got := entries[1]
	if got.Section != "API Reference > Sales" || got.Title != "Create a sales order" ||
		got.URL != "https://docs.example.com/api-reference/sales/create-sales-order.md" || got.Description != "Creates a sales order." {
		t.Errorf("unexpected entry %+v", got)
	}
}

// Entries deep in the index — the API reference — are found, which a truncated read of llms.txt misses.
func TestSearchDocEntries_RanksTitleMatches(t *testing.T) {
	t.Parallel()
	entries := parseLLMsIndex(sampleLLMsIndex)
	hits := searchDocEntries(entries, "list customers", 5)
	if len(hits) == 0 || hits[0].Title != "List customers" {
		t.Fatalf("want List customers first, got %+v", hits)
	}
	if got := searchDocEntries(entries, "zzz", 5); len(got) != 0 {
		t.Errorf("unrelated query should match nothing, got %+v", got)
	}
}

func TestAttachFullText(t *testing.T) {
	t.Parallel()
	entries := parseLLMsIndex(sampleLLMsIndex)
	attachFullText(entries, "# Overview\nSource: https://docs.example.com/get-started/overview\nInventory and fulfillment in one place.\n")
	if !strings.Contains(entries[0].Content, "fulfillment") {
		t.Errorf("full text should attach to its page, got %q", entries[0].Content)
	}
	if hits := searchDocEntries(entries, "fulfillment", 5); len(hits) != 1 || docSnippet(hits[0], "fulfillment") == "" {
		t.Errorf("body text should be searchable with a snippet, got %+v", hits)
	}
}

func docsServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/llms.txt":
			_, _ = w.Write([]byte(sampleLLMsIndex))
		case "/api-reference/sales/list-customers":
			_, _ = w.Write([]byte("# List customers\nGET /v1/sales/customers"))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

// API-reference links in the index point at .md pages that may 404; the page URL without .md is tried next.
func TestFetchDoc_FallsBackWithoutMarkdownExtension(t *testing.T) {
	t.Parallel()
	srv := docsServer(t)
	got, err := fetchDoc(context.Background(), srv.Client(), srv.URL+"/api-reference/sales/list-customers.md")
	if err != nil || !strings.Contains(got, "GET /v1/sales/customers") {
		t.Fatalf("want fallback content, got %q, %v", got, err)
	}
	if _, err := fetchDoc(context.Background(), srv.Client(), srv.URL+"/missing.md"); err == nil {
		t.Error("a page missing in both forms should error")
	}
}

func TestDocsIndexSearch(t *testing.T) {
	t.Parallel()
	srv := docsServer(t)
	x := &docsIndex{client: srv.Client(), indexURL: srv.URL + "/llms.txt", fullURL: srv.URL + "/llms-full.txt"}
	out, err := x.search(context.Background(), json.RawMessage(`{"query":"sales order"}`))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "create-sales-order.md") {
		t.Errorf("search should return the matching page URL:\n%s", out)
	}
	out, _ = x.search(context.Background(), json.RawMessage(`{"query":"zzz"}`))
	if !strings.Contains(out, "No documentation pages match") {
		t.Errorf("unexpected output %q", out)
	}
}

func TestDescribeAPIOperation(t *testing.T) {
	t.Parallel()
	runCtx := &domain.HandlerRunContext{AllowedEndpointToolSlugs: grantTools("list_customers")}
	out, err := HandleDescribeAPIOperation(context.Background(), json.RawMessage(`{"slug":"list_customers"}`), runCtx)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"GET /v1/sales/customers", "customers:read", "- include (query)", "Valid include values:", "Input schema:"} {
		if !strings.Contains(out, want) {
			t.Errorf("description missing %q", want)
		}
	}
	if !runCtx.RevealedToolSlugs["list_customers"] {
		t.Error("describing a granted operation should make it callable")
	}

	if _, err := HandleDescribeAPIOperation(context.Background(), json.RawMessage(`{"slug":"create_customer"}`), runCtx); err == nil {
		t.Error("an operation the agent isn't granted must not be described")
	}
}

// sourceArchive builds a GitHub-style tar.gz whose entries share one top-level directory.
func sourceArchive(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for name, body := range files {
		if err := tw.WriteHeader(&tar.Header{Name: "api-9.9.9/" + name, Mode: 0o644, Size: int64(len(body)), Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		_, _ = tw.Write([]byte(body))
	}
	_ = tw.Close()
	_ = gz.Close()
	return buf.Bytes()
}

func newTestSourceStore(t *testing.T, ref string) (*sourceStore, *atomic.Int32) {
	t.Helper()
	archive := sourceArchive(t, map[string]string{
		"services/core-service/internal/service/order.go":      "package service\n\n// CreditLimit blocks orders over the limit.\nfunc CheckCreditLimit() {}\n",
		"services/core-service/internal/service/order_test.go": "package service\n// CheckCreditLimit test\n",
		"services/core-service/internal/domain/mock/order.go":  "// CheckCreditLimit mock\n",
		"docs/patterns/orders.md":                              "# Orders\nCredit limits are checked on submit.\n",
		"web/app.ts":                                           "CheckCreditLimit()\n",
	})
	downloads := &atomic.Int32{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		downloads.Add(1)
		if !strings.HasSuffix(r.URL.Path, "/v9.9.9") {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write(archive)
	}))
	t.Cleanup(srv.Close)
	s := &sourceStore{
		client:     srv.Client(),
		ref:        func() string { return ref },
		archiveURL: func(ref string) string { return srv.URL + "/tar.gz/" + ref },
	}
	t.Cleanup(func() {
		if s.tree != nil {
			_ = os.RemoveAll(s.tree.dir)
		}
	})
	return s, downloads
}

func TestSourceTools_SearchAndRead(t *testing.T) {
	t.Parallel()
	s, downloads := newTestSourceStore(t, "v9.9.9")
	ctx := context.Background()

	out, err := s.search(ctx, json.RawMessage(`{"query":"checkcreditlimit"}`))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "services/core-service/internal/service/order.go:4: func CheckCreditLimit() {}") {
		t.Errorf("expected a path:line match:\n%s", out)
	}
	for _, excluded := range []string{"order_test.go", "/mock/", "app.ts"} {
		if strings.Contains(out, excluded) {
			t.Errorf("%s should be excluded from the source tree:\n%s", excluded, out)
		}
	}

	out, _ = s.search(ctx, json.RawMessage(`{"query":"credit","path_glob":"docs/**/*.md"}`))
	if !strings.Contains(out, "docs/patterns/orders.md:2:") || strings.Contains(out, "order.go") {
		t.Errorf("path_glob should restrict matches to docs:\n%s", out)
	}

	out, err = s.read(ctx, json.RawMessage(`{"path":"services/core-service/internal/service/order.go","offset":3,"limit":1}`))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "lines 3-3 of 4") || !strings.Contains(out, "CreditLimit blocks orders") {
		t.Errorf("unexpected read output:\n%s", out)
	}
	if _, err := s.read(ctx, json.RawMessage(`{"path":"../etc/passwd"}`)); err == nil {
		t.Error("paths outside the tree must be rejected")
	}
	if downloads.Load() != 1 {
		t.Errorf("the archive should be downloaded once per ref, got %d", downloads.Load())
	}
}

func TestSourceTools_UnpinnedBuildFailsClearly(t *testing.T) {
	t.Parallel()
	s, downloads := newTestSourceStore(t, "")
	_, err := s.search(context.Background(), json.RawMessage(`{"query":"x"}`))
	if err == nil || !strings.Contains(err.Error(), "not pinned") {
		t.Errorf("want a clear unpinned-build error, got %v", err)
	}
	if downloads.Load() != 0 {
		t.Error("nothing should be downloaded without a ref")
	}
}

func TestSourceTools_UnknownRefFailsAndBacksOff(t *testing.T) {
	t.Parallel()
	s, downloads := newTestSourceStore(t, "v0.0.0-missing")
	for range 2 {
		if _, err := s.read(context.Background(), json.RawMessage(`{"path":"x.go"}`)); err == nil || !strings.Contains(err.Error(), "could not download") {
			t.Errorf("want a download error, got %v", err)
		}
	}
	if downloads.Load() != 1 {
		t.Errorf("a failed download should not be retried immediately, got %d attempts", downloads.Load())
	}
}

func TestGlobRegexp(t *testing.T) {
	t.Parallel()
	cases := []struct {
		glob, path string
		want       bool
	}{
		{"services/**/*.go", "services/a/b/c.go", true},
		{"services/**/*.go", "services/c.go", true},
		{"services/*.go", "services/a/c.go", false},
		{"*.md", "README.md", true},
		{"docs/?.md", "docs/a.md", true},
	}
	for _, tc := range cases {
		re, err := globRegexp(tc.glob)
		if err != nil {
			t.Fatal(err)
		}
		if got := re.MatchString(tc.path); got != tc.want {
			t.Errorf("glob %q on %q = %v, want %v", tc.glob, tc.path, got, tc.want)
		}
	}
}
