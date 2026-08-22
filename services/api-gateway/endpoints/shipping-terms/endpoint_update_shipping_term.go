package shippingtermep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apirequest "github.com/open-mrp/api/services/api-gateway/pkg/request"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/field"
)

// Request to partially update a shipping term.
//
// Fields left out of the request keep their current values. Send an explicit JSON `null` for `flat_rate`, `minimum_order_value`, or `free_shipping_service_level_ids` to clear the stored value.
type UpdateShippingTermRequest struct {
	// Shipping term ID.
	ShippingTermID string `path:"id" validate:"required"`
	// Human-readable name for the shipping term, used to identify it when assigning shipping terms to customers and orders.
	Name field.Optional[string] `json:"name,omitzero" validate:"omitempty,max=255"`
	// Freight pricing model applied by this shipping term.
	//
	// - `free_freight`: the buyer is never charged for shipping.
	// - `flat_rate_freight`: the buyer is charged the fixed amount in `flat_rate`, regardless of what the carrier would have charged.
	// - `carrier_rate_freight`: the buyer is charged the rate the carrier quotes for the order's carrier and service level.
	Type field.Optional[constants.ShippingTermType] `json:"type,omitzero"`
	// Fixed shipping charge applied to orders.
	//
	// Used only when `type` is `flat_rate_freight`. Clearing it leaves a `flat_rate_freight` term falling through to the carrier's quoted rate.
	FlatRate field.Clearable[apirequest.QuantityInput] `json:"flat_rate,omitzero"`
	// Order total a buyer must exceed for this term's free-shipping rules to apply.
	//
	// Clearing it removes the free-shipping threshold, so orders are charged according to `type` regardless of their total.
	MinimumOrderValue field.Clearable[apirequest.QuantityInput] `json:"minimum_order_value,omitzero"`
	// IDs of the service levels that ship for free once an order exceeds `minimum_order_value`.
	//
	// Replaces the whole list rather than adding to it, and clearing it lets every service level ship free above the threshold. The request is rejected if any ID is not a service level available to your account.
	FreeShippingServiceLevelIDs field.Clearable[[]string] `json:"free_shipping_service_level_ids,omitzero"`
}

var sampleUpdateShippingTermRequest = &UpdateShippingTermRequest{
	Name:                        field.Some("Collect"),
	Type:                        field.Some(constants.ShippingTermTypeFlatRateFreight),
	FlatRate:                    field.Set(apirequest.QuantityInput{Value: "15.00", UnitID: apiresource.SampleUnitID}),
	MinimumOrderValue:           field.Set(apirequest.QuantityInput{Value: "500.00", UnitID: apiresource.SampleUnitID}),
	FreeShippingServiceLevelIDs: field.Set([]string{apiresource.SampleServiceLevelID}),
}

func (*UpdateShippingTermRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateShippingTermRequest)
}

// Partially updates a shipping term owned by your account.
//
// System-provided default shipping terms cannot be updated. Changes affect freight quoted after the update; freight already recorded on existing orders is not recalculated.
type UpdateShippingTermEndpoint struct{}

func (e *UpdateShippingTermEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateShippingTermRequest, *apiresource.ShippingTerm] {
	return (&apiendpoint.APIEndpoint[*UpdateShippingTermRequest, *apiresource.ShippingTerm]{
		Title:               "Update Shipping Term",
		Method:              http.MethodPatch,
		Route:               "/v1/operations/shipping-terms/{id}",
		ContentType:         "application/json",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainShippingTerms, Action: types.ActionUpdate}},
		Preview:             true,
		ObjectType:          constants.ObjectTypeShippingTerm,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeShippingTerm,
			Fields:     []string{"owner", "owner.account", "flat_rate.unit", "minimum_order_value.unit", "free_shipping_service_levels"},
		}),
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateShippingTermRequest) (*apiresource.ShippingTerm, *apierror.APIError) {
			return svc.(ShippingTermSvc).UpdateShippingTerm
		},
	})
}
