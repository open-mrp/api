package registrationflowep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to retrieve a registration flow.
type GetRegistrationFlowRequest struct {
	// Registration flow ID.
	RegistrationFlowID string `path:"id" validate:"required"`
}

type GetRegistrationFlowEndpoint struct{}

func (e *GetRegistrationFlowEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetRegistrationFlowRequest, *apiresource.RegistrationFlow] {
	return &apiendpoint.APIEndpoint[*GetRegistrationFlowRequest, *apiresource.RegistrationFlow]{
		Title:             "Get Registration Flow",
		Description:       "Returns a registration flow by ID.",
		Method:            http.MethodGet,
		Route:             "/v1/sales/registration-flows/{id}",
		ContentType:       "application/json",
		Request:           &GetRegistrationFlowRequest{},
		Response:          &apiresource.RegistrationFlow{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetRegistrationFlowRequest) (*apiresource.RegistrationFlow, *apierror.APIError) {
			return svc.(RegistrationFlowSvc).GetRegistrationFlow
		},
	}
}
