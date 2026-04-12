package tenancyep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// GetTenancyRequest is the request to retrieve the authenticated user's tenancy context.
type GetTenancyRequest struct{}

type GetTenancyEndpoint struct{}

func (e *GetTenancyEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetTenancyRequest, *apiresource.Tenancy] {
	return &apiendpoint.APIEndpoint[*GetTenancyRequest, *apiresource.Tenancy]{
		Title:             "Get Tenancy",
		Description:       "Returns the authenticated user's tenancy context.",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/identity/me/tenancy",
		Request:           &GetTenancyRequest{},
		Response:          &apiresource.Tenancy{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetTenancyRequest) (*apiresource.Tenancy, *apierror.APIError) {
			return svc.(TenancySvc).GetTenancy
		},
	}
}
