package versiontransforms

import (
	"net/url"

	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/version"
)

func init() {
	version.Register(&accountIntegrationForgePreview10To9{})
}

// accountIntegrationForgePreview10To9 bridges account integrations from 1.0.forge-preview.10 to 1.0.forge-preview.9.
//
// preview.10 adds `openrouter` to an integration's `provider`. A preview.9 list leaves OpenRouter integrations out, since its contract knows no such provider. A create, update or delete that names one returns it as is: only a caller that already knows the provider can reach it there.
type accountIntegrationForgePreview10To9 struct{}

func (t *accountIntegrationForgePreview10To9) FromVersion() version.APIVersion {
	return version.V1_0_Forge_Preview10
}

func (t *accountIntegrationForgePreview10To9) ToVersion() version.APIVersion {
	return version.V1_0_Forge_Preview9
}

func (t *accountIntegrationForgePreview10To9) ObjectTypes() []constants.ObjectType {
	return []constants.ObjectType{constants.ObjectTypeAccountIntegration}
}

func (t *accountIntegrationForgePreview10To9) Transform(_ constants.ObjectType, data map[string]any) map[string]any {
	if asString(data["object"]) != string(constants.ObjectTypeList) {
		return data
	}
	items, ok := data["data"].([]any)
	if !ok {
		return data
	}
	kept := make([]any, 0, len(items))
	for _, item := range items {
		if !isOpenRouterIntegration(item) {
			kept = append(kept, item)
		}
	}
	data["data"] = kept
	return data
}

func (t *accountIntegrationForgePreview10To9) TransformRequest(_ constants.ObjectType, data map[string]any) map[string]any {
	return data
}

func (t *accountIntegrationForgePreview10To9) TransformQuery(_ constants.ObjectType, _ string, query url.Values) url.Values {
	return query
}

func isOpenRouterIntegration(item any) bool {
	m, ok := item.(map[string]any)
	return ok && asString(m["object"]) == string(constants.ObjectTypeAccountIntegration) && asString(m["provider"]) == string(constants.IntegrationCodeOpenRouter)
}
