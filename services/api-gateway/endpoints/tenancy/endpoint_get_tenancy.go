package tenancyep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to retrieve the authenticated user's tenancy context.
type GetTenancyRequest struct{}

// Returns the authenticated user's tenancy context.
//
// The tenancy describes which account the user is currently acting in and every other account they can switch to, including sandboxes. It can be called before an account is selected, such as immediately after authentication; when the request does not target an account, one is chosen automatically, preferring paid accounts and then the account the user most recently used.
//
// Accounts where the user has been disabled or removed are left out, and sandboxes are only listed when the current account is a production account and the user is an administrator of it. A user who belongs to no usable account gets an empty tenancy, along with any registration they started but never finished.
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
			HideFromRequestLog: true,
		},
	})
}
