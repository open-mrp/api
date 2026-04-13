package shippingtermep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// GetShippingTermRequest is the request to retrieve a single shipping term.
type GetShippingTermRequest struct {
	// The ID of the shipping term to retrieve.
	ShippingTermID string `path:"id" validate:"required"`
}

type GetShippingTermEndpoint struct{}

func (e *GetShippingTermEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetShippingTermRequest, *apiresource.ShippingTerm] {
	return &apiendpoint.APIEndpoint[*GetShippingTermRequest, *apiresource.ShippingTerm]{
		Title:             "Get Shipping Term",
		Description:       "Returns a single shipping term by its ID.",
		Method:            http.MethodGet,
		Route:             "/v1/operations/shipping-terms/{id}",
		ContentType:       "application/json",
		Request:           &GetShippingTermRequest{},
		Response:          &apiresource.ShippingTerm{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeShippingTerm,
			Fields:     []string{"owner", "owner.account", "flat_rate.unit", "minimum_order_value.unit", "free_shipping_service_levels"},
		}),
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetShippingTermRequest) (*apiresource.ShippingTerm, *apierror.APIError) {
			return svc.(ShippingTermSvc).GetShippingTerm
		},
	}
}
