package versiontransforms

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/open-mrp/api/shared/apiversion"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/apikit/object"
	"github.com/open-mrp/apikit/version"
)

// forge flattened addresses: the nested geolocation (street_line_1, street_line_2, locality, state, postal_code,
// country) became line_1, line_2, city, state, postal_code and country on the address itself, with no geolocation
// id; receive_calendar_id became the expandable receive_calendar. Every request that takes an address uses the same
// names, address validation takes line_1 and line_2 (and its components return them), the list filters by types[]
// instead of type, and the routes moved from /v1/sales/addresses to /v1/core/addresses, with validate a POST.
//
// preview.9 callers get the geolocation back (its id null: forge has none to give), receive_calendar_id read from the
// calendar forced in, the old names in components, and their requests, routes and filter upgraded.
type addressForgeToPreview9 struct {
	forgeToPreview9
	everyObject
}

func init() {
	version.RegisterTransformer(addressForgeToPreview9{})
}

const (
	addressesRoute          = "/v1/core/addresses"
	addressRoute            = "/v1/core/addresses/{id}"
	validateAddressRoute    = "/v1/core/addresses/actions/validate"
	preview9AddressesPrefix = "/v1/sales/addresses"
)

// ForcedIncludes loads the calendar receive_calendar_id is read from, wherever preview.9 showed it.
func (addressForgeToPreview9) ForcedIncludes(objectType object.Type) []string {
	switch objectType {
	case constants.ObjectTypeAddress:
		return []string{"receive_calendar"}
	case constants.ObjectTypeInvoice, constants.ObjectTypeInvoiceForPayment:
		return []string{"billing_address.receive_calendar"}
	}
	return nil
}

func (addressForgeToPreview9) Transform(_ object.Type, data map[string]any) map[string]any {
	walkObjects(data, func(obj map[string]any) {
		switch obj["object"] {
		case string(constants.ObjectTypeAddress):
			downgradeAddress(obj)
		case string(constants.ObjectTypeAddressComponents):
			renameKey(obj, "line_1", "address_line_1")
			renameKey(obj, "line_2", "address_line_2")
		}
	})
	return data
}

func downgradeAddress(obj map[string]any) {
	if _, flat := obj["country"]; !flat {
		return
	}
	obj["geolocation"] = map[string]any{
		"id":            nil,
		"object":        string(constants.ObjectTypeGeolocation),
		"street_line_1": obj["line_1"],
		"street_line_2": obj["line_2"],
		"locality":      obj["city"],
		"state":         obj["state"],
		"postal_code":   obj["postal_code"],
		"country":       obj["country"],
	}
	for _, key := range []string{"line_1", "line_2", "city", "state", "postal_code", "country"} {
		delete(obj, key)
	}
	obj["receive_calendar_id"] = idOf(obj["receive_calendar"])
	delete(obj, "receive_calendar")
}

func (addressForgeToPreview9) TransformRequest(_ object.Type, data map[string]any) map[string]any {
	return data
}

// addressBodies names, per endpoint, the body fields that hold an address: "" is the body itself.
var addressBodies = map[string][]string{
	http.MethodPost + " " + addressesRoute:                              {""},
	http.MethodPatch + " " + addressRoute:                               {""},
	http.MethodPost + " /v1/sales/customers":                            {"bill_to_address", "ship_to_address"},
	http.MethodPost + " /v1/sales/customers/registration":               {"address"},
	http.MethodPost + " /v1/operations/suppliers":                       {"bill_to_address", "ship_to_address"},
	http.MethodPost + " /v1/operations/shipments/actions/estimate-rate": {"from_address", "to_address"},
	http.MethodPost + " /v1/operations/shipments/actions/rate-shop":     {"from_address", "to_address"},
	http.MethodPost + " /v1/sales/sales-orders":                         {"bill_to_address", "ship_to_address"},
	http.MethodPatch + " /v1/sales/sales-orders/{id}":                   {"billing_address", "shipping_address"},
	http.MethodPost + " /v1/operations/purchase-orders":                 {"bill_to_address", "ship_to_address"},
	http.MethodPatch + " /v1/operations/purchase-orders/{id}":           {"billing_address", "shipping_address"},
	http.MethodPatch + " /v1/identity/accounts/{id}":                    {"default_billing_address", "default_shipping_address"},
}

func (addressForgeToPreview9) TransformRequestBody(_ object.Type, method, route string, data map[string]any) map[string]any {
	if method == http.MethodPost && route == validateAddressRoute {
		renameKey(data, "address_line_1", "line_1")
		renameKey(data, "address_line_2", "line_2")
		return data
	}
	for _, key := range addressBodies[method+" "+route] {
		if key == "" {
			upgradeAddressBody(data)
			continue
		}
		if body, ok := data[key].(map[string]any); ok {
			upgradeAddressBody(body)
		}
	}
	return data
}

// upgradeAddressBody renames preview.9's address fields. A lowercase country, which preview.9 took, is uppercased so
// it still passes forge's ISO code check.
func upgradeAddressBody(body map[string]any) {
	renameKey(body, "street_line_1", "line_1")
	renameKey(body, "street_line_2", "line_2")
	renameKey(body, "locality", "city")
	if c, ok := body["country"].(string); ok {
		body["country"] = strings.ToUpper(c)
	}
}

func (addressForgeToPreview9) TransformQuery(_ object.Type, route string, query url.Values) url.Values {
	if route != addressesRoute {
		return query
	}
	if t := query.Get("type"); t != "" {
		query.Del("type")
		query.Add("types[]", t)
	}
	return query
}

// RoutePreview9Addresses serves preview.9's address routes from their forge paths: /v1/sales/addresses[/{id}] reach
// /v1/core/addresses[/{id}], and PUT …/actions/validate becomes the forge POST. Only a request naming preview.9 is
// rerouted. It wraps the router, since a route is matched before any endpoint's transformers run.
func RoutePreview9Addresses(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get(apiversion.Header) != apiversion.V1_0_Forge_Preview9.String() {
			next.ServeHTTP(w, r)
			return
		}
		if rest, ok := strings.CutPrefix(r.URL.Path, preview9AddressesPrefix); ok && (rest == "" || strings.HasPrefix(rest, "/")) {
			r = r.Clone(r.Context())
			r.URL.Path = addressesRoute + rest
			r.URL.RawPath = ""
		}
		if r.Method == http.MethodPut && r.URL.Path == validateAddressRoute {
			r = r.Clone(r.Context())
			r.Method = http.MethodPost
		}
		next.ServeHTTP(w, r)
	})
}
