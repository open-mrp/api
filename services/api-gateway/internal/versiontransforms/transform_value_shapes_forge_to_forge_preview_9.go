package versiontransforms

import (
	"net/url"
	"strings"
	"sync"

	"github.com/open-mrp/apikit/object"
	"github.com/open-mrp/apikit/version"

	"github.com/open-mrp/api/shared/decimalfmt"
)

// forge presents money as Money {amount, currency} and UnitPrice {amount, currency, per_unit}, and quantities and
// rates as Quantity {value, unit} and Rate {value, unit, per_unit}: value shapes with no id, object, timestamps or
// display_value, whose units are always present. preview.9 callers get the shapes they knew: a quantity or rate with
// its object type and display_value, money as a quantity or rate in the currency, and numerator_unit and
// denominator_unit. A value's row id and timestamps are not part of forge, so preview.9 responses leave them out, and
// money's currency unit is null, as it was whenever the caller did not include it.
//
// Requests are upgraded the other way: a rate's numerator_unit_id and denominator_unit_id become unit_id and
// per_unit_id, a price or amount given in a currency unit becomes {amount, currency: "usd"}, and the flat
// {field}_value fields some writes took become the field's value shape. Include keys that reached into a value's units
// are cut back to the value, or dropped when the value is no longer includable.
type valueShapesForgeToPreview9 struct {
	forgeToPreview9
	everyObject
}

func init() {
	version.RegisterTransformer(valueShapesForgeToPreview9{})
}

const preview9Currency = "usd"

// preview9ComputedFields are the values preview.9 presented as computed_quantity or computed_rate rather than as a
// stored quantity or rate, by the object type that carries them.
var preview9ComputedFields = map[string]map[string]bool{
	"item_inventory":               {"on_hand": true, "reserved": true, "available_to_promise": true, "short": true},
	"inventory_item":               {"quantity": true},
	"reconciled_item_result":       {"previous_quantity": true, "new_quantity": true},
	"purchase_order_line":          {"quantity_received": true},
	"receiving_order_line":         {"rejected_quantity": true},
	"open_orders_summary":          {"ordered": true, "back_ordered": true, "invoiced": true},
	"open_order_product":           {"quantity_ordered": true, "quantity_back_ordered": true, "quantity_invoiced": true},
	"open_order":                   {"total_ordered": true},
	"open_order_line":              {"unit_price": true, "quantity_back_ordered": true, "quantity_invoiced": true, "total_ordered": true},
	"new_customer":                 {"lifetime_revenue": true},
	"sales_totals":                 {"revenue": true, "cost": true, "quantity_invoiced": true},
	"sales_invoice":                {"revenue": true},
	"customer_pricing_finding":     {"unit_price": true, "peer_median_price": true},
	"realized_margin_finding":      {"quantity_invoiced": true, "revenue": true, "cost": true, "average_unit_price": true, "peer_median_price": true},
	"production_cost":              {"produced": true},
	"sales_order_price_quote_line": {"unit_price": true},
	"sales_order_freight_quote":    {"unit_price": true},
}

func (valueShapesForgeToPreview9) Transform(_ object.Type, data map[string]any) map[string]any {
	downgradeValues(data, "", "")
	return data
}

// downgradeValues rewrites every value shape under node, in place. owner is the object type of the nearest
// resource above node, and key the field that holds it there; a list's items keep the key of the field holding the list.
func downgradeValues(node any, owner, key string) {
	switch v := node.(type) {
	case map[string]any:
		if downgradeValue(v, preview9ComputedFields[owner][key]) {
			return
		}
		obj, _ := v["object"].(string)
		next := owner
		if obj != "" && obj != "list" {
			next = obj
		}
		for k, child := range v {
			childKey := k
			if obj == "list" && k == "data" {
				childKey = key
			}
			if s, ok := child.(string); ok && next == "sales_totals" && k == "quantity_invoiced" {
				// A sales total's quantity is summed across units, so forge sends the bare number.
				v[k] = map[string]any{"object": "computed_quantity", "value": s, "display_value": decimalfmt.Display(s, false), "unit": nil}
				continue
			}
			if k == "weighted_average_unit_cost" {
				if m, ok := child.(map[string]any); ok && isUnitPrice(m) {
					v[k] = preview9AnalyticsRate(m)
					continue
				}
			}
			downgradeValues(child, next, childKey)
		}
	case []any:
		for _, child := range v {
			downgradeValues(child, owner, key)
		}
	}
}

// downgradeValue rewrites m to its preview.9 shape when it is a value shape, and reports whether it was one.
func downgradeValue(m map[string]any, computed bool) bool {
	switch {
	case isMoney(m):
		amount, _ := m["amount"].(string)
		clear(m)
		m["value"] = amount
		m["unit"] = nil
		if computed {
			m["object"] = "computed_quantity"
			m["display_value"] = decimalfmt.Display(amount, true)
		} else {
			m["object"] = "quantity"
			m["display_value"] = "$" + decimalfmt.Display(amount, true)
		}
	case isUnitPrice(m):
		amount, _ := m["amount"].(string)
		perUnit := m["per_unit"]
		clear(m)
		m["object"] = "rate"
		if computed {
			m["object"] = "computed_rate"
		}
		m["value"] = amount
		m["numerator_unit"] = nil
		m["denominator_unit"] = perUnit
		m["display_value"] = withPer("$"+decimalfmt.Display(amount, true), perUnit)
	case isRate(m):
		value, _ := m["value"].(string)
		unit, perUnit := m["unit"], m["per_unit"]
		clear(m)
		m["object"] = "rate"
		if computed {
			m["object"] = "computed_rate"
		}
		m["value"] = value
		m["numerator_unit"] = unit
		m["denominator_unit"] = perUnit
		m["display_value"] = withPer(displayIn(value, unit), perUnit)
	case isQuantity(m):
		value, _ := m["value"].(string)
		m["object"] = "quantity"
		if computed {
			m["object"] = "computed_quantity"
		}
		m["display_value"] = displayIn(value, m["unit"])
	default:
		return false
	}
	return true
}

// preview9AnalyticsRate rewrites a cost per unit as the numerator and denominator quantities preview.9 reported it as.
func preview9AnalyticsRate(m map[string]any) map[string]any {
	amount, _ := m["amount"].(string)
	return map[string]any{
		"numerator":   map[string]any{"object": "quantity", "value": amount, "display_value": "$" + decimalfmt.Display(amount, true), "unit": nil},
		"denominator": map[string]any{"object": "quantity", "value": "1", "display_value": displayIn("1", m["per_unit"]), "unit": m["per_unit"]},
	}
}

func displayIn(value string, unit any) string {
	u, _ := unit.(map[string]any)
	abbr, _ := u["abbreviation"].(string)
	if dim, _ := u["type"].(string); dim == "currency" || u["dimension"] == "currency" {
		return abbr + decimalfmt.Display(value, true)
	}
	if abbr == "" {
		return decimalfmt.Display(value, false)
	}
	return decimalfmt.Display(value, false) + " " + abbr
}

func withPer(display string, perUnit any) string {
	u, _ := perUnit.(map[string]any)
	if abbr, _ := u["abbreviation"].(string); abbr != "" {
		return display + " / " + abbr
	}
	return display
}

// A value shape is a map without an object type whose keys are exactly the shape's.

func hasExactly(m map[string]any, keys ...string) bool {
	if len(m) != len(keys) {
		return false
	}
	for _, k := range keys {
		if _, ok := m[k]; !ok {
			return false
		}
	}
	return true
}

func isUnitObject(v any) bool {
	if v == nil {
		return true
	}
	u, ok := v.(map[string]any)
	return ok && u["object"] == "unit"
}

func isMoney(m map[string]any) bool {
	_, amount := m["amount"].(string)
	_, currency := m["currency"].(string)
	return hasExactly(m, "amount", "currency") && amount && currency
}

func isUnitPrice(m map[string]any) bool {
	_, amount := m["amount"].(string)
	_, currency := m["currency"].(string)
	return hasExactly(m, "amount", "currency", "per_unit") && amount && currency && isUnitObject(m["per_unit"])
}

func isRate(m map[string]any) bool {
	_, value := m["value"].(string)
	return hasExactly(m, "value", "unit", "per_unit") && value && isUnitObject(m["unit"]) && isUnitObject(m["per_unit"])
}

func isQuantity(m map[string]any) bool {
	_, value := m["value"].(string)
	return hasExactly(m, "value", "unit") && value && isUnitObject(m["unit"])
}

// preview9PriceFields are the request fields that took a rate with a currency numerator and now take a UnitPrice.
var preview9PriceFields = map[string]bool{"unit_price": true, "unit_cost": true, "unit_value": true, "labor_rate": true, "overhead_rate": true}

// preview9MoneyFields are the request fields that took a quantity in a currency unit and now take Money.
var preview9MoneyFields = map[string]bool{"credit_limit": true, "flat_rate": true, "minimum_order_value": true}

// preview9FlatValues are the requests' flat {prefix}_value fields that become a value shape under the prefix.
var preview9FlatValues = []string{"unit_price", "freight_amount", "amount"}

func (valueShapesForgeToPreview9) TransformRequest(_ object.Type, data map[string]any) map[string]any {
	upgradeValues(data)
	return data
}

func upgradeValues(node any) {
	switch v := node.(type) {
	case map[string]any:
		for _, prefix := range preview9FlatValues {
			upgradeFlatValue(v, prefix)
		}
		for k, child := range v {
			if m, ok := child.(map[string]any); ok {
				upgradeValue(k, m)
			}
			upgradeValues(child)
		}
	case []any:
		for _, child := range v {
			upgradeValues(child)
		}
	}
}

// upgradeValue rewrites a preview.9 rate or currency quantity held under key to its forge request shape, in place.
func upgradeValue(key string, m map[string]any) {
	if _, ok := m["numerator_unit_id"]; ok {
		value, perUnit := m["value"], m["denominator_unit_id"]
		unit := m["numerator_unit_id"]
		delete(m, "numerator_unit_id")
		delete(m, "denominator_unit_id")
		if preview9PriceFields[key] {
			delete(m, "value")
			m["amount"] = value
			m["currency"] = preview9Currency
		} else {
			m["unit_id"] = unit
		}
		m["per_unit_id"] = perUnit
		return
	}
	if _, ok := m["unit_id"]; ok && preview9MoneyFields[key] {
		m["amount"] = m["value"]
		m["currency"] = preview9Currency
		delete(m, "value")
		delete(m, "unit_id")
	}
}

// upgradeFlatValue folds {prefix}_value and its unit ids into the value shape forge takes under prefix.
func upgradeFlatValue(m map[string]any, prefix string) {
	value, ok := m[prefix+"_value"]
	if !ok {
		return
	}
	shape := map[string]any{"amount": value, "currency": preview9Currency}
	if per, ok := m[prefix+"_denominator_unit_id"]; ok {
		shape["per_unit_id"] = per
	}
	for _, suffix := range []string{"_value", "_unit_id", "_numerator_unit_id", "_denominator_unit_id"} {
		delete(m, prefix+suffix)
	}
	m[prefix] = shape
}

var (
	routeIncludesMu sync.RWMutex
	routeIncludes   = map[string]map[string]bool{}
)

// RegisterRouteIncludes records the include keys a route accepts, so preview.9 include keys can be cut back to ones
// it still offers. Call it for every endpoint as the router is built.
func RegisterRouteIncludes(route string, keys []string) {
	routeIncludesMu.Lock()
	defer routeIncludesMu.Unlock()
	set := routeIncludes[route]
	if set == nil {
		set = map[string]bool{}
		routeIncludes[route] = set
	}
	for _, k := range keys {
		set[k] = true
	}
}

func routeOffers(route, key string) bool {
	routeIncludesMu.RLock()
	defer routeIncludesMu.RUnlock()
	return routeIncludes[route][key]
}

var preview9UnitSegments = map[string]bool{"unit": true, "numerator_unit": true, "denominator_unit": true}

// preview9ValueFields name the value fields preview.9 callers could include or reach through.
var preview9ValueFields = map[string]bool{
	"quantity": true, "waste_quantity": true, "credit_limit": true, "unit_value": true, "unit_cost": true, "burn_rate": true,
	"quantity_ordered": true, "unit_price": true, "ordered_quantity": true, "default_lot": true, "freight_amount": true,
	"freight_weight": true, "flat_rate": true, "minimum_order_value": true, "amount": true, "seconds": true, "waste": true,
	"labor_rate": true, "labor_time": true, "overhead_rate": true,
}

func (valueShapesForgeToPreview9) TransformQuery(_ object.Type, route string, query url.Values) url.Values {
	for _, param := range []string{"include", "include[]"} {
		values, ok := query[param]
		if !ok {
			continue
		}
		var kept []string
		seen := map[string]bool{}
		for _, v := range values {
			for _, key := range strings.Split(v, ",") {
				key = strings.TrimSpace(key)
				if key = preview9IncludeKey(route, key); key != "" && !seen[key] {
					seen[key] = true
					kept = append(kept, key)
				}
			}
		}
		if len(kept) == 0 {
			query.Del(param)
			continue
		}
		if param == "include" {
			query[param] = []string{strings.Join(kept, ",")}
		} else {
			query[param] = kept
		}
	}
	return query
}

// preview9IncludeKey cuts a key that reached into a value's units back to the value, and drops a value key the
// route no longer offers. Other keys pass through for the endpoint to validate.
func preview9IncludeKey(route, key string) string {
	if key == "" {
		return ""
	}
	segs := strings.Split(key, ".")
	last := segs[len(segs)-1]
	if preview9UnitSegments[last] && len(segs) >= 2 && preview9ValueFields[segs[len(segs)-2]] {
		key = strings.Join(segs[:len(segs)-1], ".")
	} else if !preview9ValueFields[last] {
		return key
	}
	if routeOffers(route, key) {
		return key
	}
	return ""
}
