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

// Request to update a shipping term.
type UpdateShippingTermRequest struct {
	// Shipping term ID.
	ShippingTermID string `path:"id" validate:"required"`
	// Display name, such as `Free over $500`.
	//
	// Unique (case-insensitive) among your account's terms and the system defaults.
	Name field.Optional[string] `json:"name,omitzero" validate:"omitempty,max=255"`
	// How freight is charged.
	//
	// Changing it away from `flat_rate` clears `flat_rate`; changing it to `flat_rate` requires `flat_rate` in the same request. Changing it to `free` clears `free_shipping`.
	Type field.Optional[constants.ShippingTermType] `json:"type,omitzero"`
	// Whether the term can be assigned to customers and sales orders.
	//
	// `archived` terms stay attached to the records already using them.
	Status field.Optional[constants.ArchiveStatus] `json:"status,omitzero"`
	// Freight charged on every order.
	//
	// Required while `type` is `flat_rate`, and not allowed otherwise.
	FlatRate field.Clearable[apirequest.MoneyInput] `json:"flat_rate,omitzero"`
	// Waives freight on orders over a minimum value.
	//
	// Replaced as a whole. Send `null` to remove the threshold. Not allowed when `type` is `free`.
	FreeShipping field.Clearable[ShippingTermFreeShippingInput] `json:"free_shipping,omitzero"`
	// The free-shipping threshold as 1.0.forge-preview.9 sent it.
	MinimumOrderValue field.Clearable[apirequest.MoneyInput] `json:"minimum_order_value,omitzero" visibility:"internal"`
	// The free-shipping service levels as 1.0.forge-preview.9 sent them.
	FreeShippingServiceLevelIDs field.Clearable[[]string] `json:"free_shipping_service_level_ids,omitzero" visibility:"internal"`
}

var sampleUpdateShippingTermRequest = &UpdateShippingTermRequest{
	Name:     field.Some("Flat $15"),
	Type:     field.Some(constants.ShippingTermTypeFlatRate),
	FlatRate: field.Set(apirequest.MoneyInput{Amount: "15.00", Currency: "usd"}),
	FreeShipping: field.Set(ShippingTermFreeShippingInput{
		MinimumOrderValue: apirequest.MoneyInput{Amount: "500.00", Currency: "usd"},
	}),
}

func (*UpdateShippingTermRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateShippingTermRequest)
}

// Updates a shipping term.
//
// Only fields you send are changed. System defaults can't be updated. Freight already recorded on orders isn't recalculated.
type UpdateShippingTermEndpoint struct{}

func (e *UpdateShippingTermEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateShippingTermRequest, *apiresource.ShippingTerm] {
	return (&apiendpoint.APIEndpoint[*UpdateShippingTermRequest, *apiresource.ShippingTerm]{
		Title:             "Update Shipping Term",
		Method:            http.MethodPatch,
		Route:             "/v1/operations/shipping-terms/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainShippingTerms, Action: types.ActionUpdate}}},
		ObjectType:        constants.ObjectTypeShippingTerm,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeShippingTerm,
			Fields:     []string{"owner", "owner.account", "free_shipping.service_levels"},
		}),
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateShippingTermRequest) (*apiresource.ShippingTerm, *apierror.APIError) {
			return svc.(ShippingTermSvc).UpdateShippingTerm
		},
	})
}
