package versiontransforms

import (
	"net/http"

	"github.com/open-mrp/apikit/object"
	"github.com/open-mrp/apikit/version"
)

// forge replaced a product line's fulfillment_policy (make_to_stock | make_to_order) with supply (stock | make | buy),
// moved its default_lot to item categories, and removed its notes. preview.9 callers see fulfillment_policy derived
// from supply (a buy line has none), and default_lot and notes as null wherever a line appears. Their writes have
// fulfillment_policy turned into supply, and default_lot and notes dropped: neither is stored on a line any more.
type productLineForgeToPreview9 struct {
	forgeToPreview9
	everyObject
	noRequestChange
}

func init() {
	version.RegisterTransformer(productLineForgeToPreview9{})
}

const (
	productLinesRoute           = "/v1/catalog/product-lines"
	productLineRoute            = "/v1/catalog/product-lines/{id}"
	productLinesBulkUpsertRoute = "/v1/catalog/product-lines/actions/bulk-upsert"
)

var (
	preview9FulfillmentPolicies = map[string]string{"stock": "make_to_stock", "make": "make_to_order"}
	forgeSupplies               = invertStrings(preview9FulfillmentPolicies)
)

func (productLineForgeToPreview9) Transform(_ object.Type, data map[string]any) map[string]any {
	walkObjects(data, func(obj map[string]any) {
		if obj["object"] != "product_line" {
			return
		}
		var policy any
		if supply, ok := obj["supply"].(string); ok {
			if p, ok := preview9FulfillmentPolicies[supply]; ok {
				policy = p
			}
		}
		obj["fulfillment_policy"] = policy
		delete(obj, "supply")
		if _, ok := obj["default_lot"]; !ok {
			obj["default_lot"] = nil
		}
		if _, ok := obj["notes"]; !ok {
			obj["notes"] = nil
		}
	})
	return data
}

func (productLineForgeToPreview9) TransformRequestBody(_ object.Type, method, route string, data map[string]any) map[string]any {
	switch {
	case method == http.MethodPost && route == productLinesRoute,
		method == http.MethodPatch && route == productLineRoute:
		upgradeProductLineWrite(data)
	case method == http.MethodPost && route == productLinesBulkUpsertRoute:
		rows, _ := data["product_lines"].([]any)
		for _, row := range rows {
			if m, ok := row.(map[string]any); ok {
				delete(m, "notes")
			}
		}
	}
	return data
}

// upgradeProductLineWrite turns a preview.9 fulfillment_policy into supply and drops the fields lines no longer store.
func upgradeProductLineWrite(body map[string]any) {
	if policy, ok := body["fulfillment_policy"]; ok {
		delete(body, "fulfillment_policy")
		switch p := policy.(type) {
		case nil:
			body["supply"] = nil
		case string:
			if supply, ok := forgeSupplies[p]; ok {
				body["supply"] = supply
			} else {
				body["supply"] = p
			}
		}
	}
	delete(body, "default_lot")
	delete(body, "notes")
}
