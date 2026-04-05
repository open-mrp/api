package shippingtermep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// DeleteShippingTermRequest is the request to delete a shipping term.
type DeleteShippingTermRequest struct {
	// The ID of the shipping term to delete.
	ShippingTermID string `path:"id" validate:"required"`
}

type DeleteShippingTermEndpoint struct{}

func (e *DeleteShippingTermEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteShippingTermRequest, *apiresource.EmptyResource] {
	return &apiendpoint.APIEndpoint[*DeleteShippingTermRequest, *apiresource.EmptyResource]{
		Title:             "Delete Shipping Term",
		Description:       "Deletes an account-owned shipping term. Default shipping terms cannot be deleted.",
		Method:            http.MethodDelete,
		Route:             "/v1/operations/shipping-terms/{id}",
		ContentType:       "application/json",
		Request:           &DeleteShippingTermRequest{},
		Response:          &apiresource.EmptyResource{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteShippingTermRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(ShippingTermSvc).DeleteShippingTerm
		},
	}
}
