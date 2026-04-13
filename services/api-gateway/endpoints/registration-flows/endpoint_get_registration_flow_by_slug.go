package registrationflowep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to retrieve a registration flow by slug.
type GetRegistrationFlowBySlugRequest struct {
	// Registration flow slug.
	Slug string `path:"slug" validate:"required"`
}

type GetRegistrationFlowBySlugEndpoint struct{}

func (e *GetRegistrationFlowBySlugEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetRegistrationFlowBySlugRequest, *apiresource.RegistrationFlow] {
	return &apiendpoint.APIEndpoint[*GetRegistrationFlowBySlugRequest, *apiresource.RegistrationFlow]{
		Title:             "Get Registration Flow by Slug",
		Description:       "Returns a registration flow by slug.",
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
