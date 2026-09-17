package versiontransforms

import (
	"net/url"

	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/version"
)

func init() {
	version.Register(&inventoryChangeLogForgePreview5To4{})
}

// inventoryChangeLogForgePreview5To4 bridges the inventory change log list from 1.0.forge-preview.5 to 1.0.forge-preview.4.
//
// preview.5 bounds a list that omits `starts_at` to the 90 days before `ends_at` (or now); preview.4 searched
// the whole history, so a preview.4 list that omits `starts_at` asks for the beginning of time explicitly. The
// resource itself did not change.
type inventoryChangeLogForgePreview5To4 struct{}

const inventoryChangeLogListRoute = "/v1/operations/inventory-change-logs"

func (t *inventoryChangeLogForgePreview5To4) FromVersion() version.APIVersion {
	return version.V1_0_Forge_Preview5
}

func (t *inventoryChangeLogForgePreview5To4) ToVersion() version.APIVersion {
	return version.V1_0_Forge_Preview4
}

func (t *inventoryChangeLogForgePreview5To4) ObjectTypes() []constants.ObjectType {
	return []constants.ObjectType{constants.ObjectTypeInventoryChangeLog}
}

func (t *inventoryChangeLogForgePreview5To4) Transform(_ constants.ObjectType, data map[string]any) map[string]any {
	return data
}

func (t *inventoryChangeLogForgePreview5To4) TransformRequest(_ constants.ObjectType, data map[string]any) map[string]any {
	return data
}

func (t *inventoryChangeLogForgePreview5To4) TransformQuery(_ constants.ObjectType, route string, query url.Values) url.Values {
	return searchWholeHistoryWithoutStart(route, inventoryChangeLogListRoute, query)
}
