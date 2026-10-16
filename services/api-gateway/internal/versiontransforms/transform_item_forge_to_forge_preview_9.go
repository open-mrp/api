package versiontransforms

import (
	"net/http"
	"net/url"

	"github.com/open-mrp/api/shared/apiversion"
	"github.com/open-mrp/apikit/object"
	"github.com/open-mrp/apikit/version"
)

// forge made the item the one catalog resource: a type of material | part | product | service, a status, a supply,
// a sales section holding the selling price (unit_value → sales.unit_price) and a material section, and no notes.
// preview.9 callers see the item they knew wherever one appears: a service is a product, unit_value is the stored
// selling price (the gateway fills `sales` for preview.9 even on an item that isn't sold), notes is null, and the
// new fields are gone. Their item list keeps its starts_at / ends_at and supplier_id filters, and lists active items
// only, as archived items are the ones it used to hide as deleted.
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
var preview9ItemDropped = []string{"status", "supply", "sales", "material", "default_manufacturing_method", "stocking_unit", "default_order_unit", "order_units"}

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
	return query
}

// Preview9ItemListActiveOnly lists only active items to a preview.9 caller that names no status, as archived items are
// the ones its list hid as deleted. It wraps the router because a query transformer can't tell the list from create,
// which share the route, and create refuses a statuses parameter.
func Preview9ItemListActiveOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == itemsRoute && r.Header.Get(apiversion.Header) == apiversion.V1_0_Forge_Preview9.String() {
			query := r.URL.Query()
			if !query.Has("statuses") && !query.Has("statuses[]") {
				query.Set("statuses[]", "active")
				r = r.Clone(r.Context())
				r.URL.RawQuery = query.Encode()
			}
		}
		next.ServeHTTP(w, r)
	})
}
