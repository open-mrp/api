package versiontransforms

import (
	"net/url"
	"testing"

	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/version"
)

func pickQueryFromPreview8(route string, query url.Values) url.Values {
	return version.TransformQuery(version.V1_0_Forge_Preview8, version.V1_0_Forge_Preview9, constants.ObjectTypePick, route, query)
}

// preview.8 matched q anywhere in the pick and PO numbers, so its searches keep doing that on preview.9.
func TestPickPreview9To8_SearchMatchesAnywhere(t *testing.T) {
	query := pickQueryFromPreview8(pickListRoute, url.Values{"q": {"2410"}, "status": {"open"}})

	if got := query.Get("q_match"); got != string(constants.PickSearchMatchContains) {
		t.Errorf("q_match = %q, want %q", got, constants.PickSearchMatchContains)
	}
	if got := query.Get("q"); got != "2410" {
		t.Errorf("q must pass through, got %q", got)
	}
	if got := query.Get("status"); got != "open" {
		t.Errorf("other parameters must pass through, status = %q", got)
	}
}

func TestPickPreview9To8_ExplicitMatchIsKept(t *testing.T) {
	query := pickQueryFromPreview8(pickListRoute, url.Values{"q": {"2410"}, "q_match": {"prefix"}})

	if got := query.Get("q_match"); got != "prefix" {
		t.Errorf("q_match = %q, want the caller's own value", got)
	}
}

func TestPickPreview9To8_NoSearchIsUntouched(t *testing.T) {
	for name, query := range map[string]url.Values{
		"omitted": {"status": {"open"}},
		"empty":   {"q": {""}},
	} {
		t.Run(name, func(t *testing.T) {
			if got := pickQueryFromPreview8(pickListRoute, query); got.Has("q_match") {
				t.Errorf("a list without a search gained a q_match: %v", got)
			}
		})
	}
}

// The retrieve and actions reject parameters they do not know, so they must never gain one.
func TestPickPreview9To8_OtherRoutesAreUntouched(t *testing.T) {
	query := pickQueryFromPreview8("/v1/operations/picks/{id}", url.Values{"q": {"2410"}})

	if query.Has("q_match") {
		t.Errorf("the retrieve gained a q_match: %v", query)
	}
}

func TestPickPreview9To8_LatestIsUntouched(t *testing.T) {
	query := version.TransformQuery(version.V1_0_Forge_Preview9, version.V1_0_Forge_Preview9,
		constants.ObjectTypePick, pickListRoute, url.Values{"q": {"2410"}})

	if query.Has("q_match") {
		t.Errorf("a latest-version search must get the prefix default, not an explicit match: %v", query)
	}
}

// Older versions reach the transform through every adjacent bridge in turn.
func TestPickPreview9To8_ChainsToOlderVersions(t *testing.T) {
	query := version.TransformQuery(version.V1_0_Forge_Preview1, version.V1_0_Forge_Preview9,
		constants.ObjectTypePick, pickListRoute, url.Values{"q": {"2410"}})

	if got := query.Get("q_match"); got != string(constants.PickSearchMatchContains) {
		t.Errorf("q_match = %q, want %q for a preview.1 search", got, constants.PickSearchMatchContains)
	}
}

func TestPickPreview9To8_ResponsesPassThrough(t *testing.T) {
	single := map[string]any{"object": "pick", "id": "pk_1", "number": "2410"}
	got := version.Transform(version.V1_0_Forge_Preview9, version.V1_0_Forge_Preview8, constants.ObjectTypePick, single)
	if got["number"] != "2410" || got["id"] != "pk_1" {
		t.Errorf("the pick shape did not change between versions, got %v", got)
	}

	list := map[string]any{"object": "list", "data": []any{map[string]any{"object": "pick", "id": "pk_1"}}}
	got = version.Transform(version.V1_0_Forge_Preview9, version.V1_0_Forge_Preview8, constants.ObjectTypePick, list)
	if data, ok := got["data"].([]any); !ok || len(data) != 1 {
		t.Errorf("a list of picks must pass through, got %v", got)
	}

	if got := version.Transform(version.V1_0_Forge_Preview9, version.V1_0_Forge_Preview8, constants.ObjectTypePick, nil); got != nil {
		t.Errorf("missing data must stay missing, got %v", got)
	}
}
