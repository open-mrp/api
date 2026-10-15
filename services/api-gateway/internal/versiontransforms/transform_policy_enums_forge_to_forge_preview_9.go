package versiontransforms

import (
	"net/http"
	"net/url"

	"github.com/open-mrp/apikit/object"
	"github.com/open-mrp/apikit/version"
)

// forge shortened the commission and freight policy values shared by product lines, customers, account groups and
// freight: commission_applied | commission_exempt became applied | exempt, and billed_freight | free_freight became
// billed | free. preview.9 callers see the old values wherever a policy appears, and their writes and customer
// filters have the old values renamed.
type policyEnumsForgeToPreview9 struct {
	forgeToPreview9
	everyObject
	noRequestChange
}

func init() {
	version.RegisterTransformer(policyEnumsForgeToPreview9{})
}

var (
	preview9CommissionPolicies = map[string]string{"applied": "commission_applied", "exempt": "commission_exempt"}
	preview9FreightPolicies    = map[string]string{"billed": "billed_freight", "free": "free_freight"}
	forgeCommissionPolicies    = invertStrings(preview9CommissionPolicies)
	forgeFreightPolicies       = invertStrings(preview9FreightPolicies)
)

const (
	customersRoute       = "/v1/sales/customers"
	customersExportRoute = "/v1/sales/customers/actions/export"
)

func (policyEnumsForgeToPreview9) Transform(_ object.Type, data map[string]any) map[string]any {
	walkObjects(data, func(obj map[string]any) {
		renamePolicyValues(obj, preview9CommissionPolicies, preview9FreightPolicies)
	})
	return data
}

func (policyEnumsForgeToPreview9) TransformRequestBody(_ object.Type, method, route string, data map[string]any) map[string]any {
	if method != http.MethodPost && method != http.MethodPatch && method != http.MethodPut {
		return data
	}
	walkObjects(data, func(obj map[string]any) {
		renamePolicyValues(obj, forgeCommissionPolicies, forgeFreightPolicies)
	})
	if route == customersExportRoute {
		renameListValues(data, "commission_status_codes", forgeCommissionPolicies)
		renameListValues(data, "freight_status_codes", forgeFreightPolicies)
	}
	return data
}

func (policyEnumsForgeToPreview9) TransformQuery(_ object.Type, route string, query url.Values) url.Values {
	if route != customersRoute {
		return query
	}
	for _, key := range []string{"commission_status_codes", "commission_status_codes[]"} {
		renameQueryValues(query, key, forgeCommissionPolicies)
	}
	for _, key := range []string{"freight_status_codes", "freight_status_codes[]"} {
		renameQueryValues(query, key, forgeFreightPolicies)
	}
	return query
}

// renamePolicyValues maps the policy fields of one object: commission_policy and freight_policy anywhere, a freight
// object's policy, and a customer's freight_preferences status.
func renamePolicyValues(obj map[string]any, commission, freight map[string]string) {
	renameValue(obj, "commission_policy", commission)
	renameValue(obj, "freight_policy", freight)
	switch obj["object"] {
	case "freight":
		renameValue(obj, "policy", freight)
	case "customer_freight_preferences":
		renameValue(obj, "status", freight)
	}
}

func renameValue(obj map[string]any, key string, names map[string]string) {
	if v, ok := obj[key].(string); ok {
		if renamed, ok := names[v]; ok {
			obj[key] = renamed
		}
	}
}

func renameListValues(obj map[string]any, key string, names map[string]string) {
	values, _ := obj[key].([]any)
	for i, v := range values {
		if s, ok := v.(string); ok {
			if renamed, ok := names[s]; ok {
				values[i] = renamed
			}
		}
	}
}

func renameQueryValues(query url.Values, key string, names map[string]string) {
	values := query[key]
	for i, v := range values {
		if renamed, ok := names[v]; ok {
			values[i] = renamed
		}
	}
}

func invertStrings(m map[string]string) map[string]string {
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[v] = k
	}
	return out
}
