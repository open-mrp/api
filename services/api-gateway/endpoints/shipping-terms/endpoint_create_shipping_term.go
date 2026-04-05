package shippingtermep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// QuantityInputRequest represents a quantity value and unit pair.
type QuantityInputRequest struct {
	// The decimal value.
	Value string `json:"value" validate:"required"`
	// The unit ID for the value.
	UnitID string `json:"unit_id" validate:"required"`
}

// CreateShippingTermRequest is the request to create a new shipping term.
type CreateShippingTermRequest struct {
	// The display name of the shipping term.
	Name string `json:"name" validate:"required"`
	// The shipping term type.
	Type constants.ShippingTermType `json:"type" validate:"required"`
	// The flat rate for this shipping term.
	FlatRate *QuantityInputRequest `json:"flat_rate,omitempty"`
	// The minimum order value for free shipping under this term.
	MinimumOrderValue *QuantityInputRequest `json:"minimum_order_value,omitempty"`
	// The service level IDs that qualify for free shipping.
	FreeShippingServiceLevelIDs []string `json:"free_shipping_service_level_ids,omitempty"`
}

var sampleCreateShippingTermRequest = &CreateShippingTermRequest{
	Name:                        "Prepaid",
	Type:                        constants.ShippingTermTypeCarrierRateFreight,
	FreeShippingServiceLevelIDs: []string{},
}

func (*CreateShippingTermRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateShippingTermRequest)
}

type CreateShippingTermEndpoint struct{}

func (e *CreateShippingTermEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateShippingTermRequest, *apiresource.ShippingTerm] {
	return &apiendpoint.APIEndpoint[*CreateShippingTermRequest, *apiresource.ShippingTerm]{
		Title:             "Create Shipping Term",
		Description:       "Creates a new account-owned shipping term.",
		Method:            http.MethodPost,
		Route:             "/v1/operations/shipping-terms",
		Request:           &CreateShippingTermRequest{},
		Response:          &apiresource.ShippingTerm{},
		SuccessStatusCode: http.StatusCreated,
		Public:            true,
		Preview:           true,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeShippingTerm,
			Fields:     []string{"owner", "flat_rate.unit", "minimum_order_value.unit", "free_shipping_service_levels"},
		}),
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateShippingTermRequest) (*apiresource.ShippingTerm, *apierror.APIError) {
			return svc.(ShippingTermSvc).CreateShippingTerm
		},
	}
}
