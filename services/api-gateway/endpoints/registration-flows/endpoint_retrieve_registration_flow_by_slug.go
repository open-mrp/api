package registrationflowep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to retrieve a registration flow by slug.
type RetrieveRegistrationFlowBySlugRequest struct {
	// Registration flow slug.
	Slug string `path:"slug" validate:"required"`
}

// Returns a registration flow by slug.
type RetrieveRegistrationFlowBySlugEndpoint struct{}

func (e *RetrieveRegistrationFlowBySlugEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveRegistrationFlowBySlugRequest, *apiresource.RegistrationFlow] {
	return (&apiendpoint.APIEndpoint[*RetrieveRegistrationFlowBySlugRequest, *apiresource.RegistrationFlow]{
		Title:             "Retrieve Registration Flow by Slug",
		Method:            http.MethodGet,
		Route:             "/v1/sales/registration-flows/by-slug/{slug}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeRegistrationFlow,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveRegistrationFlowBySlugRequest) (*apiresource.RegistrationFlow, *apierror.APIError) {
			return svc.(RegistrationFlowSvc).GetRegistrationFlowBySlug
		},
	})
}
