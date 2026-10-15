package shippingtermep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apirequest "github.com/open-mrp/api/services/api-gateway/pkg/request"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
	apiexample "github.com/open-mrp/apikit/example"
	"github.com/open-mrp/apikit/field"
)

// When a shipping term waives freight.
type ShippingTermFreeShippingInput struct {
	// Order total that must be exceeded for freight to be waived.
	MinimumOrderValue apirequest.MoneyInput `json:"minimum_order_value" validate:"required"`
	// Service levels that qualify for free freight. Omit to waive freight on every service level.
	ServiceLevelIDs []string `json:"service_level_ids,omitzero"`
}

// Request to create a shipping term.
type CreateShippingTermRequest struct {
	// Display name, such as `Free over $500`.
	//
	// Unique (case-insensitive) among your account's terms and the system defaults.
	Name string `json:"name" validate:"required,max=255"`
	// How freight is charged.
	//
	// `free`: never charged. `flat_rate`: charged `flat_rate` on every order. `carrier_rate`: charged the carrier's quoted rate for the order's service level.
	Type constants.ShippingTermType `json:"type" validate:"required"`
	// Initial status.
	//
	// Defaults to `active`; create as `archived` to set a term up before offering it.
	Status field.Optional[constants.ArchiveStatus] `json:"status,omitzero"`
	// Freight charged on every order.
	//
	// Required when `type` is `flat_rate`, and not allowed otherwise.
	FlatRate field.Optional[apirequest.MoneyInput] `json:"flat_rate,omitzero"`
	// Waives freight on orders over a minimum value.
	//
	// Omit for no threshold. Not allowed when `type` is `free`.
	FreeShipping field.Optional[ShippingTermFreeShippingInput] `json:"free_shipping,omitzero"`
	// The free-shipping threshold as 1.0.forge-preview.9 sent it.
	MinimumOrderValue field.Optional[apirequest.MoneyInput] `json:"minimum_order_value,omitzero" visibility:"internal"`
	// The free-shipping service levels as 1.0.forge-preview.9 sent them.
	FreeShippingServiceLevelIDs []string `json:"free_shipping_service_level_ids,omitzero" visibility:"internal"`
}

var sampleCreateShippingTermRequest = &CreateShippingTermRequest{
	Name: "Free over $500",
	Type: constants.ShippingTermTypeCarrierRate,
	FreeShipping: field.Some(ShippingTermFreeShippingInput{
		MinimumOrderValue: apirequest.MoneyInput{Amount: "500.00", Currency: "usd"},
		ServiceLevelIDs:   []string{apiresource.SampleServiceLevelID},
	}),
}

func (*CreateShippingTermRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateShippingTermRequest)
}

// Creates a shipping term.
//
// The new term belongs to your account. It affects freight quotes once it's assigned to a customer or sales order.
type CreateShippingTermEndpoint struct{}

func (e *CreateShippingTermEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateShippingTermRequest, *apiresource.ShippingTerm] {
	return (&apiendpoint.APIEndpoint[*CreateShippingTermRequest, *apiresource.ShippingTerm]{
		Title:             "Create Shipping Term",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/operations/shipping-terms",
		SuccessStatusCode: http.StatusCreated,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainShippingTerms, Action: types.ActionCreate}}},
		ObjectType:        constants.ObjectTypeShippingTerm,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeShippingTerm,
			Fields:     []string{"owner", "owner.account", "free_shipping.service_levels"},
		}),
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateShippingTermRequest) (*apiresource.ShippingTerm, *apierror.APIError) {
			return svc.(ShippingTermSvc).CreateShippingTerm
		},
		LocationFunc: func(resp *apiresource.ShippingTerm) string {
			return "/v1/operations/shipping-terms/" + resp.ID
		},
	})
}
