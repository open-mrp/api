package shippingtermep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apirequest "github.com/augno/api/services/api-gateway/pkg/request"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// Request to create a shipping term.
type CreateShippingTermRequest struct {
	// Human-readable name for the shipping term, used to identify it when assigning shipping terms to customers and orders.
	Name string `json:"name" validate:"required,max=255"`
	// Freight pricing model applied by this shipping term.
	//
	// - `free_freight`: the buyer is never charged for shipping.
	// - `flat_rate_freight`: the buyer is charged the fixed amount in `flat_rate`, regardless of what the carrier would have charged.
	// - `carrier_rate_freight`: the buyer is charged the rate the carrier quotes for the order's carrier and service level.
	Type constants.ShippingTermType `json:"type" validate:"required"`
	// Fixed shipping charge applied to orders.
	//
	// Used only when `type` is `flat_rate_freight`. A `flat_rate_freight` term created without a flat rate falls through to the carrier's quoted rate.
	FlatRate field.Optional[apirequest.QuantityInput] `json:"flat_rate,omitzero"`
	// Order total a buyer must exceed for this term's free-shipping rules to apply.
	//
	// Above this total, freight is free for the service levels in `free_shipping_service_level_ids`.
	MinimumOrderValue field.Optional[apirequest.QuantityInput] `json:"minimum_order_value,omitzero"`
	// IDs of the service levels that ship for free once an order exceeds `minimum_order_value`.
	//
	// Leave this empty to let every service level ship free above the threshold. The request is rejected if any ID is not a service level available to your account.
	FreeShippingServiceLevelIDs []string `json:"free_shipping_service_level_ids,omitzero"`
}

var sampleCreateShippingTermRequest = &CreateShippingTermRequest{
	Name:                        "Prepaid",
	Type:                        constants.ShippingTermTypeFlatRateFreight,
	FlatRate:                    field.Some(apirequest.QuantityInput{Value: "15.00", UnitID: apiresource.SampleUnitID}),
	MinimumOrderValue:           field.Some(apirequest.QuantityInput{Value: "500.00", UnitID: apiresource.SampleUnitID}),
	FreeShippingServiceLevelIDs: []string{apiresource.SampleServiceLevelID},
}

func (*CreateShippingTermRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateShippingTermRequest)
}

// Creates a shipping term owned by your account.
//
// The new term takes effect on freight quoting once it is assigned as a customer's default shipping term.
type CreateShippingTermEndpoint struct{}

func (e *CreateShippingTermEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateShippingTermRequest, *apiresource.ShippingTerm] {
	return (&apiendpoint.APIEndpoint[*CreateShippingTermRequest, *apiresource.ShippingTerm]{
		Title:               "Create Shipping Term",
		Method:              http.MethodPost,
		ContentType:         "application/json",
		Route:               "/v1/operations/shipping-terms",
		SuccessStatusCode:   http.StatusCreated,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainShippingTerms, Action: types.ActionCreate}},
		Preview:             true,
		ObjectType:          constants.ObjectTypeShippingTerm,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeShippingTerm,
			Fields:     []string{"owner", "owner.account", "flat_rate.unit", "minimum_order_value.unit", "free_shipping_service_levels"},
		}),
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateShippingTermRequest) (*apiresource.ShippingTerm, *apierror.APIError) {
			return svc.(ShippingTermSvc).CreateShippingTerm
		},
		LocationFunc: func(resp *apiresource.ShippingTerm) string {
			return "/v1/operations/shipping-terms/" + resp.ID
		},
	})
}
