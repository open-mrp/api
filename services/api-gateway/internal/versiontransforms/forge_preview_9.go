package versiontransforms

import (
	"github.com/open-mrp/api/shared/apiversion"
	"github.com/open-mrp/apikit/object"
	"github.com/open-mrp/apikit/version"
)

// forgeToPreview9 is the version pair every transformer here converts between: responses built for 1.0.forge are
// downgraded to 1.0.forge-preview.9, and preview.9 requests are upgraded.
type forgeToPreview9 struct{}

func (forgeToPreview9) FromVersion() version.APIVersion { return apiversion.V1_0_Forge }
func (forgeToPreview9) ToVersion() version.APIVersion   { return apiversion.V1_0_Forge_Preview9 }

// everyObject applies to every response and request, whatever the endpoint's object type.
type everyObject struct{}

func (everyObject) ObjectTypes() []object.Type { return []object.Type{version.AnyObjectType} }

// noRequestChange leaves preview.9 request bodies as they are.
type noRequestChange struct{}

func (noRequestChange) TransformRequest(_ object.Type, data map[string]any) map[string]any {
	return data
}

// walkObjects calls fn on every JSON object in node, depth first, parents before children.
func walkObjects(node any, fn func(map[string]any)) {
	switch v := node.(type) {
	case map[string]any:
		fn(v)
		for _, child := range v {
			walkObjects(child, fn)
		}
	case []any:
		for _, child := range v {
			walkObjects(child, fn)
		}
	}
}
