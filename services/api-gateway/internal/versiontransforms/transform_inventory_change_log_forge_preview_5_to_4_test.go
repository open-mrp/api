package versiontransforms

import (
	"net/url"
	"testing"

	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/version"
)

func TestInventoryChangeLogPreview5To4_OmittedStartSearchesEverything(t *testing.T) {
	query := version.TransformQuery(version.V1_0_Forge_Preview4, version.V1_0_Forge_Preview5,
		constants.ObjectTypeInventoryChangeLog, inventoryChangeLogListRoute, url.Values{})
	if got := query.Get("starts_at"); got != unboundedStart {
		t.Errorf("starts_at = %q, want %q", got, unboundedStart)
	}
}

func TestInventoryChangeLogPreview5To4_ExplicitStartIsKept(t *testing.T) {
	query := version.TransformQuery(version.V1_0_Forge_Preview4, version.V1_0_Forge_Preview5,
		constants.ObjectTypeInventoryChangeLog, inventoryChangeLogListRoute, url.Values{"starts_at": {"2026-09-01T00:00:00Z"}})
	if got := query.Get("starts_at"); got != "2026-09-01T00:00:00Z" {
		t.Errorf("starts_at = %q, want the caller's own value", got)
	}
}

func TestInventoryChangeLogPreview5To4_OtherRoutesAreUntouched(t *testing.T) {
	query := version.TransformQuery(version.V1_0_Forge_Preview4, version.V1_0_Forge_Preview5,
		constants.ObjectTypeInventoryChangeLog, inventoryChangeLogListRoute+"/{id}", url.Values{})
	if query.Has("starts_at") {
		t.Errorf("a non-list route gained a starts_at: %v", query)
	}
}
