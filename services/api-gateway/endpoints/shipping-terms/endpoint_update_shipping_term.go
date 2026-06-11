package shippingtermep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apirequest "github.com/augno/api/services/api-gateway/pkg/request"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// Request to partially update a shipping term.
//
// All fields are optional and absent fields are left unchanged. Send an explicit JSON `null` for `flat_rate`, `minimum_order_value`, or `free_shipping_service_level_ids` to clear the existing value.
type UpdateShippingTermRequest struct {
	// Shipping term ID.
	ShippingTermID string `path:"id" validate:"required"`
	// Human-readable name for the shipping term, used to identify it when assigning shipping terms to customers and orders.
	Name field.Optional[string] `json:"name,omitzero" validate:"omitempty,max=255"`
	// Freight pricing model applied by this shipping term.
	//
	// - `free_freight`: no shipping cost to the buyer.
	// - `flat_rate_freight`: a fixed shipping cost regardless of order details (see `flat_rate`).
	// - `carrier_rate_freight`: shipping cost is determined by the carrier's quoted rate.
	Type field.Optional[constants.ShippingTermType] `json:"type,omitzero"`
	// Fixed shipping charge applied to orders.
	//
	// Only applied when `type` is `flat_rate_freight`. Send `null` to clear.
	FlatRate field.Clearable[apirequest.QuantityInput] `json:"flat_rate,omitzero"`
	// Order subtotal a buyer must reach for this term's free-shipping rules to apply.
	//
	// Send `null` to clear.
	MinimumOrderValue field.Clearable[apirequest.QuantityInput] `json:"minimum_order_value,omitzero"`
	// IDs of service levels that ship for free under this term (typically once `minimum_order_value` is met).
	//
	// Replaces the existing list. Send `null` to clear.
	FreeShippingServiceLevelIDs field.Clearable[[]string] `json:"free_shipping_service_level_ids,omitzero"`
}

var sampleUpdateShippingTermRequest = &UpdateShippingTermRequest{
	Name: field.Some("Collect"),
}

func (*UpdateShippingTermRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateShippingTermRequest)
}

// Partially updates an account-owned shipping term.
//
// System-provided default shipping terms cannot be updated.
type UpdateShippingTermEndpoint struct{}

func (e *UpdateShippingTermEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateShippingTermRequest, *apiresource.ShippingTerm] {
	return (&apiendpoint.APIEndpoint[*UpdateShippingTermRequest, *apiresource.ShippingTerm]{
		Title:             "Update Shipping Term",
		Method:            http.MethodPatch,
		Route:             "/v1/operations/shipping-terms/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ObjectType:        constants.ObjectTypeShippingTerm,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeShippingTerm,
			Fields:     []string{"owner", "owner.account", "flat_rate.unit", "minimum_order_value.unit", "free_shipping_service_levels"},
		}),
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateShippingTermRequest) (*apiresource.ShippingTerm, *apierror.APIError) {
			return svc.(ShippingTermSvc).UpdateShippingTerm
		},
	})
}
