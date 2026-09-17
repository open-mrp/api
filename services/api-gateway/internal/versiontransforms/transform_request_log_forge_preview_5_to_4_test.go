package versiontransforms

import (
	"net/url"
	"testing"

	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/version"
)

// preview.4 searched every log when starts_at was omitted, so its lists keep doing that on preview.5.
func TestRequestLogPreview5To4_OmittedStartSearchesEverything(t *testing.T) {
	query := version.TransformQuery(version.V1_0_Forge_Preview4, version.V1_0_Forge_Preview5,
		constants.ObjectTypeRequestLog, requestLogListRoute, url.Values{"methods": {"GET"}})

	if got := query.Get("starts_at"); got != unboundedStart {
		t.Errorf("starts_at = %q, want %q", got, unboundedStart)
	}
	if got := query.Get("methods"); got != "GET" {
		t.Errorf("other parameters must pass through, methods = %q", got)
	}
}

func TestRequestLogPreview5To4_ExplicitStartIsKept(t *testing.T) {
	query := version.TransformQuery(version.V1_0_Forge_Preview4, version.V1_0_Forge_Preview5,
		constants.ObjectTypeRequestLog, requestLogListRoute, url.Values{"starts_at": {"2026-09-01T00:00:00Z"}})

	if got := query.Get("starts_at"); got != "2026-09-01T00:00:00Z" {
		t.Errorf("starts_at = %q, want the caller's own value", got)
	}
}

// The retrieve rejects unknown parameters, so it must never gain a window.
func TestRequestLogPreview5To4_RetrieveIsUntouched(t *testing.T) {
	query := version.TransformQuery(version.V1_0_Forge_Preview4, version.V1_0_Forge_Preview5,
		constants.ObjectTypeRequestLog, "/v1/core/request-logs/{id}", url.Values{})

	if query.Has("starts_at") {
		t.Errorf("the retrieve gained a starts_at: %v", query)
	}
}

func TestRequestLogPreview5To4_LatestIsUntouched(t *testing.T) {
	query := version.TransformQuery(version.V1_0_Forge_Preview5, version.V1_0_Forge_Preview5,
		constants.ObjectTypeRequestLog, requestLogListRoute, url.Values{})

	if query.Has("starts_at") {
		t.Errorf("a latest-version list must get the service's default window, not an explicit start: %v", query)
	}
}

func TestRequestLogPreview5To4_ResponsesPassThrough(t *testing.T) {
	data := map[string]any{"object": "request_log", "id": "rq_1", "path": "/v1/catalog/items"}
	got := version.Transform(version.V1_0_Forge_Preview5, version.V1_0_Forge_Preview4, constants.ObjectTypeRequestLog, data)
	if got["path"] != "/v1/catalog/items" || got["id"] != "rq_1" {
		t.Errorf("the request log shape did not change between versions, got %v", got)
	}
}
