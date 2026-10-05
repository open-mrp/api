package versiontransforms

import (
	"net/http"

	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/version"
)

func init() {
	version.Register(&registrationFlowForgePreview6To5{})
}

// registrationFlowForgePreview6To5 bridges registration flow updates from 1.0.forge-preview.6 to 1.0.forge-preview.5.
//
// preview.5 replaced an option list only when its `has_<list>` flag was true, and ignored the list otherwise. preview.6 replaces a list whenever it is sent; an omitted list is unchanged. Responses and the create body did not change.
type registrationFlowForgePreview6To5 struct{}

const registrationFlowUpdateRoute = "/v1/sales/registration-flows/{id}"

// registrationFlowOptionLists pairs each preview.5 flag with the list it governed.
var registrationFlowOptionLists = []struct{ flag, list string }{
	{"has_customer_group_ids", "customer_group_ids"},
	{"has_payment_term_ids", "payment_term_ids"},
	{"has_shipping_term_ids", "shipping_term_ids"},
}

func (t *registrationFlowForgePreview6To5) FromVersion() version.APIVersion {
	return version.V1_0_Forge_Preview6
}

func (t *registrationFlowForgePreview6To5) ToVersion() version.APIVersion {
	return version.V1_0_Forge_Preview5
}

func (t *registrationFlowForgePreview6To5) ObjectTypes() []constants.ObjectType {
	return []constants.ObjectType{constants.ObjectTypeRegistrationFlow}
}

func (t *registrationFlowForgePreview6To5) Transform(_ constants.ObjectType, data map[string]any) map[string]any {
	// The registration flow resource did not change.
	return data
}

func (t *registrationFlowForgePreview6To5) TransformRequest(_ constants.ObjectType, data map[string]any) map[string]any {
	// Superseded by TransformRequestBody, which knows which endpoint the body is for.
	return data
}

// TransformRequestBody turns a preview.5 update's flags into the preview.6 meaning: a flagged list is sent (an absent or null one meant "clear", so it becomes an empty list), and an unflagged one is dropped.
func (t *registrationFlowForgePreview6To5) TransformRequestBody(_ constants.ObjectType, method, route string, data map[string]any) map[string]any {
	if method != http.MethodPatch || route != registrationFlowUpdateRoute {
		return data
	}
	for _, pair := range registrationFlowOptionLists {
		raw, present := data[pair.flag]
		if !present {
			delete(data, pair.list)
			continue
		}
		replace, isBool := raw.(bool)
		if !isBool {
			// Not a preview.5 flag; leave it for the decoder to reject.
			continue
		}
		delete(data, pair.flag)
		if !replace {
			delete(data, pair.list)
			continue
		}
		if ids, ok := data[pair.list]; !ok || ids == nil {
			data[pair.list] = []any{}
		}
	}
	return data
}
