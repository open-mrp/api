package versiontransforms

import (
	"github.com/open-mrp/apikit/object"
	"github.com/open-mrp/apikit/version"
)

// forge added the `site` location type; each account's top-level location became one. preview.9 callers see a site
// as the `building` it was, wherever a location appears. Their writes can't name `site`, so requests pass through.
type locationForgeToPreview9 struct {
	forgeToPreview9
	everyObject
	noRequestChange
}

func init() {
	version.RegisterTransformer(locationForgeToPreview9{})
}

func (locationForgeToPreview9) Transform(_ object.Type, data map[string]any) map[string]any {
	walkObjects(data, func(obj map[string]any) {
		if obj["object"] == "location" && obj["type"] == "site" {
			obj["type"] = "building"
		}
	})
	return data
}
