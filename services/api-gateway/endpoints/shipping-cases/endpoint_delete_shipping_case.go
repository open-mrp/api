package shippingcaseep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// DeleteShippingCaseRequest is the request to delete a shipping case.
type DeleteShippingCaseRequest struct {
	// The ID of the shipping case to delete.
	ShippingCaseID string `path:"id" validate:"required"`
}

type DeleteShippingCaseEndpoint struct{}

func (e *DeleteShippingCaseEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteShippingCaseRequest, *apiresource.EmptyResource] {
	return &apiendpoint.APIEndpoint[*DeleteShippingCaseRequest, *apiresource.EmptyResource]{
		Title:             "Delete Shipping Case",
		Description:       "Permanently deletes a shipping case.",
		Method:            http.MethodDelete,
		Route:             "/v1/operations/shipping-cases/{id}",
		ContentType:       "application/json",
		Request:           &DeleteShippingCaseRequest{},
		Response:          &apiresource.EmptyResource{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteShippingCaseRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(ShippingCaseSvc).DeleteShippingCase
		},
	}
}
