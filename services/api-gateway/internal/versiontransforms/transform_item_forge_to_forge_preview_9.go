package versiontransforms

import (
	"net/url"

	"github.com/open-mrp/apikit/object"
	"github.com/open-mrp/apikit/version"
)

// forge made the item the one catalog resource: a type of material | part | product | service, a status, a supply,
// a sales section holding the selling price (unit_value → sales.unit_price) and a material section, and no notes.
// preview.9 callers see the item they knew wherever one appears: a service is a product, unit_value is the sales
// price (null for an item that isn't sold), notes is null, and the new fields are gone. Their item list keeps its
// starts_at / ends_at and supplier_id filters, and lists active items only, as archived items are the ones it used to
// hide as deleted.
type itemForgeToPreview9 struct {
	forgeToPreview9
	everyObject
	noRequestChange
}

func init() {
	version.RegisterTransformer(itemForgeToPreview9{})
}

const itemsRoute = "/v1/catalog/items"

// preview9ItemDropped are the item fields preview.9 never had.
var preview9ItemDropped = []string{"status", "supply", "sales", "material", "default_manufacturing_method"}

func (itemForgeToPreview9) Transform(_ object.Type, data map[string]any) map[string]any {
	walkObjects(data, func(obj map[string]any) {
		if obj["object"] != "item" {
			return
		}
		downgradeItem(obj)
	})
	return data
}

// downgradeItem rewrites one forge item to its preview.9 shape, in place.
func downgradeItem(item map[string]any) {
	if item["type"] == "service" {
		item["type"] = "product"
	}
	if sales, ok := item["sales"]; ok {
		var unitValue any
		if s, ok := sales.(map[string]any); ok {
			unitValue = s["unit_price"]
		}
		item["unit_value"] = unitValue
	}
	if _, ok := item["sku"]; ok {
		item["notes"] = nil
	}
	for _, k := range preview9ItemDropped {
		delete(item, k)
	}
}

func (itemForgeToPreview9) TransformRequestBody(_ object.Type, _, _ string, data map[string]any) map[string]any {
	return data
}

func (itemForgeToPreview9) TransformQuery(_ object.Type, route string, query url.Values) url.Values {
	if route != itemsRoute {
		return query
	}
	for from, to := range map[string]string{"starts_at": "created_after", "ends_at": "created_before"} {
		if v := query.Get(from); v != "" {
			query.Del(from)
			query.Set(to, v)
		}
	}
	if v := query.Get("supplier_id"); v != "" {
		query.Del("supplier_id")
		query.Add("supplier_ids[]", v)
	}
	if !query.Has("statuses") && !query.Has("statuses[]") {
		query.Set("statuses[]", "active")
	}
	return query
}
