package versiontransforms

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/open-mrp/apikit/object"
	"github.com/open-mrp/apikit/version"
)

// forge renamed a shipping term's types to free | flat_rate | carrier_rate and gathered its minimum order value and
// free-shipping service levels into a free_shipping section, whose service levels are included as
// free_shipping.service_levels. preview.9 callers see the old type names and the two top-level fields wherever a
// shipping term appears; their writes have the type renamed (the separate threshold fields are still accepted), and
// their free_shipping_service_levels include key is renamed.
type shippingTermForgeToPreview9 struct {
	forgeToPreview9
	everyObject
	noRequestChange
}

func init() {
	version.RegisterTransformer(shippingTermForgeToPreview9{})
}

const (
	shippingTermsRoute = "/v1/operations/shipping-terms"
	shippingTermRoute  = "/v1/operations/shipping-terms/{id}"

	preview9FreeShippingLevelsKey = "free_shipping_service_levels"
	forgeFreeShippingLevelsKey    = "free_shipping.service_levels"
)

// preview9ShippingTermTypes maps each forge type to the name preview.9 used.
var preview9ShippingTermTypes = map[string]string{
	"free":         "free_freight",
	"flat_rate":    "flat_rate_freight",
	"carrier_rate": "carrier_rate_freight",
}

func (shippingTermForgeToPreview9) Transform(_ object.Type, data map[string]any) map[string]any {
	walkObjects(data, func(obj map[string]any) {
		if obj["object"] != "shipping_term" {
			return
		}
		if t, ok := obj["type"].(string); ok {
			if old, renamed := preview9ShippingTermTypes[t]; renamed {
				obj["type"] = old
			}
		}
		if _, ok := obj["free_shipping"]; !ok {
			return
		}
		fs, _ := obj["free_shipping"].(map[string]any)
		delete(obj, "free_shipping")
		obj["minimum_order_value"] = fs["minimum_order_value"]
		obj[preview9FreeShippingLevelsKey] = fs["service_levels"]
	})
	return data
}

func (shippingTermForgeToPreview9) TransformRequestBody(_ object.Type, method, route string, data map[string]any) map[string]any {
	if (method == http.MethodPost && route == shippingTermsRoute) || (method == http.MethodPatch && route == shippingTermRoute) {
		if t, ok := data["type"].(string); ok {
			for forge, old := range preview9ShippingTermTypes {
				if t == old {
					data["type"] = forge
				}
			}
		}
	}
	return data
}

// TransformQuery renames the free_shipping_service_levels include key, at the end of any include path, on routes
// that offer the forge key.
func (shippingTermForgeToPreview9) TransformQuery(_ object.Type, route string, query url.Values) url.Values {
	for _, param := range []string{"include", "include[]"} {
		values, ok := query[param]
		if !ok {
			continue
		}
		renamed := make([]string, len(values))
		for i, v := range values {
			keys := strings.Split(v, ",")
			for j, key := range keys {
				key = strings.TrimSpace(key)
				if key == preview9FreeShippingLevelsKey || strings.HasSuffix(key, "."+preview9FreeShippingLevelsKey) {
					forgeKey := strings.TrimSuffix(key, preview9FreeShippingLevelsKey) + forgeFreeShippingLevelsKey
					if routeOffers(route, forgeKey) {
						key = forgeKey
					}
				}
				keys[j] = key
			}
			renamed[i] = strings.Join(keys, ",")
		}
		query[param] = renamed
	}
	return query
}
