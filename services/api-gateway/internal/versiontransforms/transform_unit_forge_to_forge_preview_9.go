package versiontransforms

import (
	"net/http"
	"net/url"

	"github.com/open-mrp/apikit/object"
	"github.com/open-mrp/apikit/version"
)

// forge renamed a unit's type to dimension, removed is_base_unit, and filters the list by dimensions[] instead of
// type. preview.9 callers see type and is_base_unit wherever a unit appears (base units are the platform's system
// units); their creates and bulk upserts have type renamed and is_base_unit dropped, and their type filter becomes
// dimensions[].
type unitForgeToPreview9 struct {
	forgeToPreview9
	everyObject
	noRequestChange
}

func init() {
	version.RegisterTransformer(unitForgeToPreview9{})
}

const (
	unitsRoute           = "/v1/catalog/units"
	unitsBulkUpsertRoute = "/v1/catalog/units/actions/bulk-upsert"
)

// preview9BaseUnitIDs are the system units each dimension converts through; no other unit is a base unit.
var preview9BaseUnitIDs = map[string]bool{
	"celcius": true, "dollar": true, "each": true, "gram": true, "hour": true, "liter": true, "meter": true,
	"meters_squared": true,
}

func (unitForgeToPreview9) Transform(_ object.Type, data map[string]any) map[string]any {
	walkObjects(data, func(obj map[string]any) {
		if obj["object"] != "unit" {
			return
		}
		if d, ok := obj["dimension"]; ok {
			delete(obj, "dimension")
			obj["type"] = d
		}
		id, _ := obj["id"].(string)
		obj["is_base_unit"] = preview9BaseUnitIDs[id]
	})
	return data
}

func (unitForgeToPreview9) TransformRequestBody(_ object.Type, method, route string, data map[string]any) map[string]any {
	if method != http.MethodPost {
		return data
	}
	switch route {
	case unitsRoute:
		upgradeUnitWrite(data)
	case unitsBulkUpsertRoute:
		rows, _ := data["units"].([]any)
		for _, row := range rows {
			if m, ok := row.(map[string]any); ok {
				upgradeUnitWrite(m)
			}
		}
	}
	return data
}

// upgradeUnitWrite renames a preview.9 type to dimension and drops is_base_unit, which forge no longer accepts.
func upgradeUnitWrite(body map[string]any) {
	if t, ok := body["type"]; ok {
		delete(body, "type")
		body["dimension"] = t
	}
	delete(body, "is_base_unit")
}

func (unitForgeToPreview9) TransformQuery(_ object.Type, route string, query url.Values) url.Values {
	if route != unitsRoute {
		return query
	}
	if t := query.Get("type"); t != "" {
		query.Del("type")
		query.Add("dimensions[]", t)
	}
	return query
}
