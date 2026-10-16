package versiontransforms

import (
	"net/http"
	"net/url"

	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/apikit/object"
	"github.com/open-mrp/apikit/version"
)

// forge reshaped suppliers like customers: they gained status (active | archived), note became instructions,
// contact, carrier_billing and defaults { carrier, service_level, payment_term, shipping_term } are new, and
// bill_to_address / ship_to_address moved under defaults. material_count left the public shape; preview.9 callers
// still read it.
//
// preview.9 callers get the old shape back, and their writes and list filters upgraded. Their update's note and
// update_note pair becomes instructions, and their default address ids move under defaults. The include keys
// bill_to_address and ship_to_address are upgraded by the customer transformer, which maps them for any route.
type supplierForgeToPreview9 struct {
	forgeToPreview9
	everyObject
	noRequestChange
}

func init() {
	version.RegisterTransformer(supplierForgeToPreview9{})
}

const (
	suppliersRoute = "/v1/operations/suppliers"
	supplierRoute  = "/v1/operations/suppliers/{id}"
)

func (supplierForgeToPreview9) Transform(_ object.Type, data map[string]any) map[string]any {
	walkObjects(data, func(obj map[string]any) {
		if obj["object"] == string(constants.ObjectTypeSupplier) {
			downgradeSupplier(obj)
		}
	})
	return data
}

func downgradeSupplier(obj map[string]any) {
	defaults, forge := obj["defaults"].(map[string]any)
	if !forge {
		return
	}
	renameKey(obj, "instructions", "note")
	obj["bill_to_address"] = defaults["bill_to_address"]
	obj["ship_to_address"] = defaults["ship_to_address"]
	if _, ok := obj["material_count"]; !ok {
		obj["material_count"] = nil
	}
	for _, key := range []string{"status", "contact", "carrier_billing", "defaults"} {
		delete(obj, key)
	}
}

func (supplierForgeToPreview9) TransformRequestBody(_ object.Type, method, route string, data map[string]any) map[string]any {
	switch {
	case method == http.MethodPost && route == suppliersRoute:
		renameKey(data, "note", "instructions")
	case method == http.MethodPatch && route == supplierRoute:
		upgradeSupplierUpdate(data)
	}
	return data
}

// upgradeSupplierUpdate turns preview.9's note / update_note pair into instructions (a note applied without a value
// clears them) and moves its default address ids under defaults.
func upgradeSupplierUpdate(body map[string]any) {
	note, hasNote := body["note"]
	apply, _ := body["update_note"].(bool)
	delete(body, "note")
	delete(body, "update_note")
	if apply {
		if !hasNote {
			note = nil
		}
		body["instructions"] = note
	}

	defaults := map[string]any{}
	for _, key := range []string{"bill_to_address_id", "ship_to_address_id"} {
		if v, ok := body[key]; ok {
			defaults[key] = v
			delete(body, key)
		}
	}
	if len(defaults) > 0 {
		body["defaults"] = defaults
	}
}

// preview9SupplierFilters are preview.9's supplier list filters that forge renamed.
var preview9SupplierFilters = map[string]string{
	"starts_at": "created_after",
	"ends_at":   "created_before",
}

func (supplierForgeToPreview9) TransformQuery(_ object.Type, route string, query url.Values) url.Values {
	if route != suppliersRoute {
		return query
	}
	for old, forge := range preview9SupplierFilters {
		if values, ok := query[old]; ok {
			query.Del(old)
			query[forge] = values
		}
	}
	return query
}
