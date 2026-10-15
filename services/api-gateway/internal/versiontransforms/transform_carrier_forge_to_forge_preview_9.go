package versiontransforms

import (
	"net/http"
	"net/url"

	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/apikit/object"
	"github.com/open-mrp/apikit/version"
)

// forge reshaped carriers and service levels:
//   - carrier: the optional code became a required type (ltl1 → ltl, delivery → local_delivery, none or
//     freight_collect → other), customer_portal_visibility became portal_visibility, deleted_at went away, and the
//     carrier names its default_service_level instead of each service level flagging is_default.
//   - service level: service_level_token became code, default_transit_days became estimated_transit_days, and
//     customer_portal_visibility became portal_visibility.
//
// preview.9 callers get the old names back, each service level's is_default derived from its carrier's default
// (forced in so it is there to read), and their writes renamed. A service level create or update still takes
// is_default, which forge keeps as an internal field.
type carrierForgeToPreview9 struct {
	forgeToPreview9
	everyObject
}

func init() {
	version.RegisterTransformer(carrierForgeToPreview9{})
}

const (
	carriersRoute      = "/v1/operations/carriers"
	carrierRoute       = "/v1/operations/carriers/{id}"
	serviceLevelsRoute = "/v1/operations/carriers/{carrier_id}/service-levels"
	serviceLevelRoute  = "/v1/operations/carriers/{carrier_id}/service-levels/{id}"
)

// ForcedIncludes loads what is_default is derived from: a carrier's default service level, or a service level's
// carrier and that carrier's default.
func (carrierForgeToPreview9) ForcedIncludes(objectType object.Type) []string {
	switch objectType {
	case constants.ObjectTypeCarrier:
		return []string{"default_service_level"}
	case constants.ObjectTypeServiceLevel:
		return []string{"carrier", "carrier.default_service_level"}
	case constants.ObjectTypeCustomer:
		return []string{"freight_preferences.carrier.default_service_level"}
	}
	return nil
}

func (carrierForgeToPreview9) Transform(_ object.Type, data map[string]any) map[string]any {
	walkObjects(data, func(obj map[string]any) {
		switch obj["object"] {
		case string(constants.ObjectTypeCarrier):
			downgradeCarrier(obj)
		case string(constants.ObjectTypeServiceLevel):
			downgradeServiceLevel(obj)
		}
	})
	return data
}

// downgradeCarrier runs before the carrier's own service levels are walked, so it marks their is_default while the
// default service level is still on the carrier.
func downgradeCarrier(obj map[string]any) {
	if t, ok := obj["type"].(string); ok {
		obj["code"] = preview9CarrierCode(t)
		delete(obj, "type")
	}
	renameKey(obj, "portal_visibility", "customer_portal_visibility")
	obj["deleted_at"] = nil

	defaultID := idOf(obj["default_service_level"])
	if levels, ok := obj["service_levels"].(map[string]any); ok {
		data, _ := levels["data"].([]any)
		for _, item := range data {
			if sl, ok := item.(map[string]any); ok {
				sl["is_default"] = defaultID != nil && sl["id"] == defaultID
			}
		}
	}
	delete(obj, "default_service_level")
}

func downgradeServiceLevel(obj map[string]any) {
	renameKey(obj, "code", "service_level_token")
	renameKey(obj, "estimated_transit_days", "default_transit_days")
	renameKey(obj, "portal_visibility", "customer_portal_visibility")
	if _, set := obj["is_default"]; !set {
		var defaultID any
		if carrier, ok := obj["carrier"].(map[string]any); ok {
			defaultID = idOf(carrier["default_service_level"])
		}
		obj["is_default"] = defaultID != nil && obj["id"] == defaultID
	}
	delete(obj, "carrier")
}

// preview9CarrierCode is the code preview.9 knew a carrier type by; `other` is the absence of one.
func preview9CarrierCode(t string) any {
	switch constants.CarrierType(t) {
	case constants.CarrierTypeLocalDelivery:
		return "delivery"
	case constants.CarrierTypeOther:
		return nil
	default:
		return t
	}
}

func renameKey(obj map[string]any, from, to string) {
	if v, ok := obj[from]; ok {
		obj[to] = v
		delete(obj, from)
	}
}

func (carrierForgeToPreview9) TransformRequest(_ object.Type, data map[string]any) map[string]any {
	return data
}

func (carrierForgeToPreview9) TransformRequestBody(_ object.Type, method, route string, data map[string]any) map[string]any {
	switch {
	case method == http.MethodPost && route == carriersRoute:
		data["type"] = forgeCarrierType(data["code"])
		delete(data, "code")
		renameKey(data, "customer_portal_visibility", "portal_visibility")
	case method == http.MethodPatch && route == carrierRoute:
		renameKey(data, "customer_portal_visibility", "portal_visibility")
	case (method == http.MethodPost && route == serviceLevelsRoute) || (method == http.MethodPatch && route == serviceLevelRoute):
		renameKey(data, "customer_portal_visibility", "portal_visibility")
		renameKey(data, "default_transit_days", "estimated_transit_days")
	}
	return data
}

// forgeCarrierType is the type a preview.9 carrier code became. A code forge doesn't know passes through, so it
// fails validation as before.
func forgeCarrierType(code any) any {
	c, ok := code.(string)
	if !ok || c == "" {
		return string(constants.CarrierTypeOther)
	}
	switch c {
	case "ltl1":
		return string(constants.CarrierTypeLTL)
	case "delivery":
		return string(constants.CarrierTypeLocalDelivery)
	case "freight_collect":
		return string(constants.CarrierTypeOther)
	default:
		return c
	}
}

func (carrierForgeToPreview9) TransformQuery(_ object.Type, _ string, query url.Values) url.Values {
	return query
}
