package versiontransforms

import (
	"testing"

	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/version"
)

func accountIntegrationToPreview9(data map[string]any) map[string]any {
	return version.Transform(version.V1_0_Forge_Preview10, version.V1_0_Forge_Preview9, constants.ObjectTypeAccountIntegration, data)
}

func integrationObject(id, provider string) map[string]any {
	return map[string]any{"id": id, "object": "account_integration", "provider": provider, "monthly_spend_limit": nil}
}

func TestAccountIntegrationPreview10To9_ListDropsOpenRouter(t *testing.T) {
	got := accountIntegrationToPreview9(map[string]any{
		"object":   "list",
		"has_more": false,
		"data": []any{
			integrationObject("acig_stripe", "stripe"),
			integrationObject("acig_openrouter", "openrouter"),
			integrationObject("acig_shippo", "shippo"),
		},
	})

	items := got["data"].([]any)
	if len(items) != 2 {
		t.Fatalf("got %d integrations, want 2: %v", len(items), items)
	}
	for _, item := range items {
		if item.(map[string]any)["provider"] == "openrouter" {
			t.Errorf("a preview.9 list carried an OpenRouter integration: %v", item)
		}
	}
	if got["has_more"] != false {
		t.Errorf("the list's other fields must pass through, has_more = %v", got["has_more"])
	}
}

// A caller only reaches a single OpenRouter integration by naming the provider or its ID itself.
func TestAccountIntegrationPreview10To9_SingleObjectIsUntouched(t *testing.T) {
	got := accountIntegrationToPreview9(integrationObject("acig_openrouter", "openrouter"))

	if got["provider"] != "openrouter" || got["id"] != "acig_openrouter" {
		t.Errorf("a single integration must pass through, got %v", got)
	}
}

func TestAccountIntegrationPreview10To9_ListWithoutDataIsUntouched(t *testing.T) {
	got := accountIntegrationToPreview9(map[string]any{"object": "list", "has_more": false})

	if _, ok := got["data"]; ok {
		t.Errorf("a list without data gained one: %v", got)
	}
}

func TestAccountIntegrationPreview10To9_LatestIsUntouched(t *testing.T) {
	data := map[string]any{"object": "list", "data": []any{integrationObject("acig_openrouter", "openrouter")}}

	got := version.Transform(version.V1_0_Forge_Preview10, version.V1_0_Forge_Preview10, constants.ObjectTypeAccountIntegration, data)

	if len(got["data"].([]any)) != 1 {
		t.Errorf("a preview.10 caller must keep its OpenRouter integration, got %v", got)
	}
}
