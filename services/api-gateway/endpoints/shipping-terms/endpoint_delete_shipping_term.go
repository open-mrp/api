package shippingtermep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to delete a shipping term.
type DeleteShippingTermRequest struct {
	// Shipping term ID.
	ShippingTermID string `path:"id" validate:"required"`
}

// Deletes an account-owned shipping term.
//
// System-provided default shipping terms cannot be deleted.
type DeleteShippingTermEndpoint struct{}

func (e *DeleteShippingTermEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteShippingTermRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*DeleteShippingTermRequest, *apiresource.EmptyResource]{
		Title:             "Delete Shipping Term",
		Method:            http.MethodDelete,
		Route:             "/v1/operations/shipping-terms/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteShippingTermRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(ShippingTermSvc).DeleteShippingTerm
		},
	})
}
