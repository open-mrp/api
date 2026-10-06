package versiontransforms

import (
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/version"
)

func init() {
	version.Register(&accountUserForgePreview6To5{})
}

// accountUserForgePreview6To5 downgrades account users from 1.0.forge-preview.6 to 1.0.forge-preview.5, wherever one appears in a response.
//
// preview.6 withholds an account user's `is_commission_eligible` and `last_used_at` from customer and supplier portal users. `last_used_at` was already nullable; `is_commission_eligible` became nullable where preview.5 always carried a boolean, so a preview.5 portal caller reads it as false. The seller's own users read the real flag in both versions. Requests did not change.
type accountUserForgePreview6To5 struct{}

var accountUserForgePreview6To5Withheld = map[string]any{"is_commission_eligible": false}

func (t *accountUserForgePreview6To5) FromVersion() version.APIVersion {
	return version.V1_0_Forge_Preview6
}

func (t *accountUserForgePreview6To5) ToVersion() version.APIVersion {
	return version.V1_0_Forge_Preview5
}

func (t *accountUserForgePreview6To5) ObjectTypes() []constants.ObjectType {
	return everyObjectType()
}

func (t *accountUserForgePreview6To5) Transform(_ constants.ObjectType, data map[string]any) map[string]any {
	zeroWithheldIn(data, constants.ObjectTypeAccountUser, accountUserForgePreview6To5Withheld)
	return data
}

func (t *accountUserForgePreview6To5) TransformRequest(_ constants.ObjectType, data map[string]any) map[string]any {
	return data
}
