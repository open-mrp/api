package tenancyep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to retrieve the authenticated user's tenancy context.
type GetTenancyRequest struct{}

// Returns the authenticated user's tenancy context.
type GetTenancyEndpoint struct{}

func (e *GetTenancyEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetTenancyRequest, *apiresource.Tenancy] {
	return (&apiendpoint.APIEndpoint[*GetTenancyRequest, *apiresource.Tenancy]{
		Title:             "Get Tenancy",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/identity/me/tenancy",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeTenancy,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetTenancyRequest) (*apiresource.Tenancy, *apierror.APIError) {
			return svc.(TenancySvc).GetTenancy
		},
		Extras: apiendpoint.APIEndpointExtras{
			SkipRequestLogging: true,
		},
	})
}
