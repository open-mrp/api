package versiontransforms

import (
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/version"
)

func init() {
	version.Register(&accountGroupForgePreview6To5{})
}

// accountGroupForgePreview6To5 downgrades account groups from 1.0.forge-preview.6 to 1.0.forge-preview.5, wherever one appears in a response.
//
// preview.6 withholds an account group's `commission_policy` from customer and supplier portal users, so it became nullable where preview.5 always carried one of its values. A preview.5 portal caller reads it as ""; the seller's own users read the real policy in both versions. Requests did not change.
type accountGroupForgePreview6To5 struct{}

var accountGroupForgePreview6To5Withheld = map[string]any{"commission_policy": ""}

func (t *accountGroupForgePreview6To5) FromVersion() version.APIVersion {
	return version.V1_0_Forge_Preview6
}

func (t *accountGroupForgePreview6To5) ToVersion() version.APIVersion {
	return version.V1_0_Forge_Preview5
}

func (t *accountGroupForgePreview6To5) ObjectTypes() []constants.ObjectType {
	return everyObjectType()
}

func (t *accountGroupForgePreview6To5) Transform(_ constants.ObjectType, data map[string]any) map[string]any {
	zeroWithheldIn(data, constants.ObjectTypeAccountGroup, accountGroupForgePreview6To5Withheld)
	return data
}

func (t *accountGroupForgePreview6To5) TransformRequest(_ constants.ObjectType, data map[string]any) map[string]any {
	return data
}
