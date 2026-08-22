package httptransport

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
)

func TestBindFromQuery_embeddedPaginationRequest(t *testing.T) {
	t.Parallel()

	type listReq struct {
		apiresource.PaginationRequest
		Statuses []string `query:"statuses"`
	}

	u := mustParseURL(t, "/items?cursor=c1&limit=50&q=search&statuses=a&statuses=b")
	dst := &listReq{}
	if err := BindFromQuery(u, dst); err != nil {
		t.Fatal(err)
	}
	if dst.Cursor == nil || *dst.Cursor != "c1" {
		t.Fatalf("cursor: got %+v", dst.Cursor)
	}
	if dst.Limit != 50 {
		t.Fatalf("limit: got %d", dst.Limit)
	}
	if dst.Query == nil || *dst.Query != "search" {
		t.Fatalf("query: got %+v", dst.Query)
	}
	if len(dst.Statuses) != 2 || dst.Statuses[0] != "a" || dst.Statuses[1] != "b" {
		t.Fatalf("statuses: got %#v", dst.Statuses)
	}

	if apiErr := RejectUnknownQueryParams(u, dst, false); apiErr != nil {
		t.Fatalf("RejectUnknownQueryParams: %v", apiErr)
	}
}

func TestBindFromQuery_optionalPointerScalarsAllocate(t *testing.T) {
	t.Parallel()

	type req struct {
		Q      *string    `query:"q"`
		Since  *time.Time `query:"since" time_layout:"2006-01-02"`
		Absent *string    `query:"absent"`
	}

	u := mustParseURL(t, "/x?q=needle&since=2025-05-18")
	dst := &req{}
	if err := BindFromQuery(u, dst); err != nil {
		t.Fatal(err)
	}
	if dst.Q == nil || *dst.Q != "needle" {
		t.Fatalf("Q: %+v", dst.Q)
	}
	wantSince := time.Date(2025, 5, 18, 0, 0, 0, 0, time.UTC)
	if dst.Since == nil || !dst.Since.Equal(wantSince) {
		t.Fatalf("Since: got %+v want %v", dst.Since, wantSince)
	}
	if dst.Absent != nil {
		t.Fatalf("Absent: expected nil when query missing, got %v", dst.Absent)
	}
}

func TestPlanFor_cache_returnsSamePlan(t *testing.T) {
	t.Parallel()

	type listReq struct {
		apiresource.PaginationRequest
		Statuses []string `query:"statuses"`
	}

	p1, err := planFor(&listReq{})
	if err != nil {
		t.Fatal(err)
	}
	p2, err := planFor(&listReq{})
	if err != nil {
		t.Fatal(err)
	}
	if p1 != p2 {
		t.Fatal("expected cached bind plan pointer to be stable per type")
	}
}

func BenchmarkBindFromQuery(b *testing.B) {
	type listReq struct {
		apiresource.PaginationRequest
		Statuses []string `query:"statuses" default:"active,expired,revoked"`
	}
	u, err := url.Parse("/v1/items?cursor=c1&limit=50&q=needle&statuses=a&statuses=b&include=role")
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		dst := &listReq{}
		if err := BindFromQuery(u, dst); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRejectUnknownQueryParams(b *testing.B) {
	type listReq struct {
		apiresource.PaginationRequest
		Statuses []string `query:"statuses" default:"active,expired,revoked"`
	}
	u, err := url.Parse("/v1/items?cursor=c1&limit=50&q=needle&statuses=a&statuses=b&include=role")
	if err != nil {
		b.Fatal(err)
	}
	dst := &listReq{}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if apiErr := RejectUnknownQueryParams(u, dst, true); apiErr != nil {
			b.Fatal(apiErr)
		}
	}
}

func BenchmarkBindIncomingRequest(b *testing.B) {
	type listReq struct {
		apiresource.PaginationRequest
		Statuses []string `query:"statuses" default:"active,expired,revoked"`
	}
	raw := "/v1/items?cursor=c1&limit=50&q=needle&statuses=a&statuses=b&include=role"
	req := httptest.NewRequest(http.MethodGet, raw, nil)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		dst := &listReq{}
		if err := BindIncomingRequest(req, dst, true); err != nil {
			b.Fatal(err)
		}
	}
}
