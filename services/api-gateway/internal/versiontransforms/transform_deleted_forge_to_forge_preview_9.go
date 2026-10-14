package versiontransforms

import (
	"github.com/open-mrp/apikit/object"
	"github.com/open-mrp/apikit/version"
)

// forge answers a DELETE with the deleted stub {id, object, deleted: true}. preview.9 answered with an empty object.
type deletedForgeToPreview9 struct {
	forgeToPreview9
	everyObject
	noRequestChange
}

func init() {
	version.RegisterTransformer(deletedForgeToPreview9{})
}

func (deletedForgeToPreview9) Transform(_ object.Type, data map[string]any) map[string]any {
	if len(data) == 3 && data["deleted"] == true && data["id"] != nil && data["object"] != nil {
		return map[string]any{}
	}
	return data
}
