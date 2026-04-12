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
)

// UpdateShippingTermRequest is the request to partially update a shipping term.
// All fields are optional. Absent fields are left unchanged. Send an explicit
// JSON null for flat_rate, minimum_order_value, or free_shipping_service_level_ids
// to clear the existing value.
type UpdateShippingTermRequest struct {
	// The ID of the shipping term to update.
	ShippingTermID string `path:"id" validate:"required"`
	// The display name of the shipping term.
	Name *string `json:"name,omitempty" nullable:"false" validate:"omitempty,max=255"`
	// The shipping term type.
	Type *constants.ShippingTermType `json:"type,omitempty" nullable:"false"`
	// The flat rate for this shipping term. Send null to clear.
	FlatRate apirequest.NullableInput[QuantityInputRequest] `json:"flat_rate,omitempty"`
	// The minimum order value for free shipping under this term. Send null to clear.
	MinimumOrderValue apirequest.NullableInput[QuantityInputRequest] `json:"minimum_order_value,omitempty"`
	// The service level IDs that qualify for free shipping. Send null to clear.
	FreeShippingServiceLevelIDs apirequest.NullableInput[[]string] `json:"free_shipping_service_level_ids,omitempty"`
}

var sampleUpdateShippingTermRequest = &UpdateShippingTermRequest{
	Name: new("Collect"),
}

func (*UpdateShippingTermRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateShippingTermRequest)
}

type UpdateShippingTermEndpoint struct{}

func (e *UpdateShippingTermEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateShippingTermRequest, *apiresource.ShippingTerm] {
	return &apiendpoint.APIEndpoint[*UpdateShippingTermRequest, *apiresource.ShippingTerm]{
		Title:             "Update Shipping Term",
		Description:       "Partially updates an account-owned shipping term. Default shipping terms cannot be updated.",
		Method:            http.MethodPatch,
		Route:             "/v1/operations/shipping-terms/{id}",
		ContentType:       "application/json",
		Request:           &UpdateShippingTermRequest{},
		Response:          &apiresource.ShippingTerm{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeShippingTerm,
			Fields:     []string{"owner", "flat_rate.unit", "minimum_order_value.unit", "free_shipping_service_levels"},
		}),
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateShippingTermRequest) (*apiresource.ShippingTerm, *apierror.APIError) {
			return svc.(ShippingTermSvc).UpdateShippingTerm
		},
	}
}
