package regsessionep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Returns a paginated list of the authenticated user's registration sessions that are still in progress, newest first.
//
// The list is empty once the user has finished registering.
type ListSessionsEndpoint struct{}

func (e *ListSessionsEndpoint) Materialize() *apiendpoint.APIEndpoint[*apiresource.PaginationRequest, *apiresource.List[apiresource.RegistrationSession]] {
	return (&apiendpoint.APIEndpoint[*apiresource.PaginationRequest, *apiresource.List[apiresource.RegistrationSession]]{
		Title:             "List Registration Sessions",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/auth/registration-sessions",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *apiresource.PaginationRequest) (*apiresource.List[apiresource.RegistrationSession], *apierror.APIError) {
			return svc.(RegistrationSessionSvc).ListSessions
		},
	})
}
