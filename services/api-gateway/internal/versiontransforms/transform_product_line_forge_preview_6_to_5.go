package versiontransforms

import (
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/version"
)

func init() {
	version.Register(&productLineForgePreview6To5{})
}

// productLineForgePreview6To5 downgrades product lines from 1.0.forge-preview.6 to 1.0.forge-preview.5, wherever one appears in a response.
//
// preview.6 withholds a product line's `commission_policy` and `notes` from customer and supplier portal users. `notes` was already nullable; `commission_policy` became nullable where preview.5 always carried one of its values, so a preview.5 portal caller reads it as "". The seller's own users read the real policy in both versions. Requests did not change.
type productLineForgePreview6To5 struct{}

var productLineForgePreview6To5Withheld = map[string]any{"commission_policy": ""}

func (t *productLineForgePreview6To5) FromVersion() version.APIVersion {
	return version.V1_0_Forge_Preview6
}

func (t *productLineForgePreview6To5) ToVersion() version.APIVersion {
	return version.V1_0_Forge_Preview5
}

func (t *productLineForgePreview6To5) ObjectTypes() []constants.ObjectType {
	return everyObjectType()
}

func (t *productLineForgePreview6To5) Transform(_ constants.ObjectType, data map[string]any) map[string]any {
	zeroWithheldIn(data, constants.ObjectTypeProductLine, productLineForgePreview6To5Withheld)
	return data
}

func (t *productLineForgePreview6To5) TransformRequest(_ constants.ObjectType, data map[string]any) map[string]any {
	return data
}
