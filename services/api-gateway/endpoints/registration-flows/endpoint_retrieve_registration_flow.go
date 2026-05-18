package registrationflowep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to retrieve a registration flow.
type RetrieveRegistrationFlowRequest struct {
	// Registration flow ID.
	RegistrationFlowID string `path:"id" validate:"required"`
}

// Returns a registration flow by ID.
type RetrieveRegistrationFlowEndpoint struct{}

func (e *RetrieveRegistrationFlowEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveRegistrationFlowRequest, *apiresource.RegistrationFlow] {
	return (&apiendpoint.APIEndpoint[*RetrieveRegistrationFlowRequest, *apiresource.RegistrationFlow]{
		Title:             "Retrieve Registration Flow",
		Method:            http.MethodGet,
		Route:             "/v1/sales/registration-flows/{id}",
		ContentType:       "application/json",
		Request:           &RetrieveRegistrationFlowRequest{},
		Response:          &apiresource.RegistrationFlow{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveRegistrationFlowRequest) (*apiresource.RegistrationFlow, *apierror.APIError) {
			return svc.(RegistrationFlowSvc).GetRegistrationFlow
		},
	}).WithDocSource(e)
}
