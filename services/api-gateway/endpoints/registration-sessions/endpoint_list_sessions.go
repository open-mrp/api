package regsessionep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Returns a paginated list of open registration sessions for the authenticated user.
type ListSessionsEndpoint struct{}

func (e *ListSessionsEndpoint) Materialize() *apiendpoint.APIEndpoint[*apiresource.PaginationRequest, *apiresource.List[apiresource.RegistrationSession]] {
	return (&apiendpoint.APIEndpoint[*apiresource.PaginationRequest, *apiresource.List[apiresource.RegistrationSession]]{
		Title:             "List Registration Sessions",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/auth/registration-sessions",
		Request:           &apiresource.PaginationRequest{},
		Response:          &apiresource.List[apiresource.RegistrationSession]{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *apiresource.PaginationRequest) (*apiresource.List[apiresource.RegistrationSession], *apierror.APIError) {
			return svc.(RegistrationSessionSvc).ListSessions
		},
	}).WithDocSource(e)
}
