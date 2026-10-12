package versiontransforms

import (
	"net/url"

	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/version"
)

func init() {
	version.Register(&pickForgePreview9To8{})
}

// pickForgePreview9To8 bridges the pick list's search from 1.0.forge-preview.9 to 1.0.forge-preview.8.
//
// preview.9 matches `q` against the start of the pick number and the customer PO number, unless
// `q_match=contains` is sent. preview.8 matched anywhere in them, so a preview.8 search asks for that
// explicitly. The resource itself did not change.
type pickForgePreview9To8 struct{}

// pickListRoute is the only pick endpoint that searches; the others reject parameters they do not know.
const pickListRoute = "/v1/operations/picks"

func (t *pickForgePreview9To8) FromVersion() version.APIVersion {
	return version.V1_0_Forge_Preview9
}

func (t *pickForgePreview9To8) ToVersion() version.APIVersion {
	return version.V1_0_Forge_Preview8
}

func (t *pickForgePreview9To8) ObjectTypes() []constants.ObjectType {
	return []constants.ObjectType{constants.ObjectTypePick}
}

func (t *pickForgePreview9To8) Transform(_ constants.ObjectType, data map[string]any) map[string]any {
	return data
}

func (t *pickForgePreview9To8) TransformRequest(_ constants.ObjectType, data map[string]any) map[string]any {
	return data
}

func (t *pickForgePreview9To8) TransformQuery(_ constants.ObjectType, route string, query url.Values) url.Values {
	if route != pickListRoute || query.Get("q") == "" || query.Has("q_match") {
		return query
	}
	query.Set("q_match", string(constants.PickSearchMatchContains))
	return query
}
