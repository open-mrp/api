package versiontransforms

import (
	"github.com/open-mrp/apikit/object"
	"github.com/open-mrp/apikit/version"
)

// forge renamed a payment term's `inactive` status to `archived`. preview.9 callers see `inactive` wherever a payment
// term appears; they never sent a status, so their requests pass through.
type paymentTermForgeToPreview9 struct {
	forgeToPreview9
	everyObject
	noRequestChange
}

func init() {
	version.RegisterTransformer(paymentTermForgeToPreview9{})
}

func (paymentTermForgeToPreview9) Transform(_ object.Type, data map[string]any) map[string]any {
	walkObjects(data, func(obj map[string]any) {
		if obj["object"] == "payment_term" && obj["status"] == "archived" {
			obj["status"] = "inactive"
		}
	})
	return data
}
