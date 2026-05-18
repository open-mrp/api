package registrationflowep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to delete a registration flow.
type DeleteRegistrationFlowRequest struct {
	// Registration flow ID.
	RegistrationFlowID string `path:"id" validate:"required"`
}

// Deletes a registration flow.
type DeleteRegistrationFlowEndpoint struct{}

func (e *DeleteRegistrationFlowEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteRegistrationFlowRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*DeleteRegistrationFlowRequest, *apiresource.EmptyResource]{
		Title:             "Delete Registration Flow",
		Method:            http.MethodDelete,
		Route:             "/v1/sales/registration-flows/{id}",
		ContentType:       "application/json",
		Request:           &DeleteRegistrationFlowRequest{},
		Response:          &apiresource.EmptyResource{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteRegistrationFlowRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(RegistrationFlowSvc).DeleteRegistrationFlow
		},
	}).WithDocSource(e)
}
