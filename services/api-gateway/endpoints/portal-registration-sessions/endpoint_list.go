package portalregsessionep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list the account's portal registration sessions.
type ListPortalRegistrationSessionsRequest struct {
	apiresource.PaginationRequest
	// Restrict the results to a single lifecycle status.
	Status *string `query:"status"`
}

// Returns the account's buyer customer-portal registration sessions, newest first, so customer service can follow up on registrations that stalled or expired before completing. Includes in-progress, completed, abandoned, and expired sessions; filter with `status`.
type ListPortalRegistrationSessionsEndpoint struct{}

func (e *ListPortalRegistrationSessionsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListPortalRegistrationSessionsRequest, *apiresource.List[apiresource.PortalRegistrationSession]] {
	return (&apiendpoint.APIEndpoint[*ListPortalRegistrationSessionsRequest, *apiresource.List[apiresource.PortalRegistrationSession]]{
		Title:             "List Portal Registration Sessions",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/sales/portal-registration-sessions",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainAccount, Action: types.ActionRead},
		},
		ObjectType: constants.ObjectTypePortalRegistrationSession,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListPortalRegistrationSessionsRequest) (*apiresource.List[apiresource.PortalRegistrationSession], *apierror.APIError) {
			return svc.(PortalRegistrationSessionSvc).List
		},
	})
}
