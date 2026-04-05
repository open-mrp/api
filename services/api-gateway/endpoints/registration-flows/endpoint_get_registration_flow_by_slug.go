package registrationflowep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// GetRegistrationFlowBySlugRequest is the request to retrieve a single registration flow by its slug.
type GetRegistrationFlowBySlugRequest struct {
	// The slug of the registration flow to retrieve.
	Slug string `path:"slug" validate:"required"`
}

type GetRegistrationFlowBySlugEndpoint struct{}

func (e *GetRegistrationFlowBySlugEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetRegistrationFlowBySlugRequest, *apiresource.RegistrationFlow] {
	return &apiendpoint.APIEndpoint[*GetRegistrationFlowBySlugRequest, *apiresource.RegistrationFlow]{
		Title:             "Get Registration Flow by Slug",
		Description:       "Returns a single registration flow by its slug.",
		Method:            http.MethodGet,
		Route:             "/v1/sales/registration-flows/by-slug/{slug}",
		ContentType:       "application/json",
		Request:           &GetRegistrationFlowBySlugRequest{},
		Response:          &apiresource.RegistrationFlow{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetRegistrationFlowBySlugRequest) (*apiresource.RegistrationFlow, *apierror.APIError) {
			return svc.(RegistrationFlowSvc).GetRegistrationFlowBySlug
		},
	}
}
