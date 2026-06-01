package shippingtermep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to retrieve a shipping term.
type RetrieveShippingTermRequest struct {
	// Shipping term ID.
	ShippingTermID string `path:"id" validate:"required"`
}

// Returns a shipping term by ID.
type RetrieveShippingTermEndpoint struct{}

func (e *RetrieveShippingTermEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveShippingTermRequest, *apiresource.ShippingTerm] {
	return (&apiendpoint.APIEndpoint[*RetrieveShippingTermRequest, *apiresource.ShippingTerm]{
		Title:             "Retrieve Shipping Term",
		Method:            http.MethodGet,
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
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveShippingTermRequest) (*apiresource.ShippingTerm, *apierror.APIError) {
			return svc.(ShippingTermSvc).GetShippingTerm
		},
	})
}
