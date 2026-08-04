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
	// Restrict the results to a single registration state.
	//
	// - `in_progress`: still incomplete and inside the seven-day resume window.
	// - `completed`: the buyer finished registering.
	// - `abandoned`: the buyer explicitly gave the session up.
	// - `expired`: still incomplete, but past the resume window, so the buyer can no longer pick it back up.
	Status *string `query:"status"`
}

// Returns the account's buyer registrations into its customer portal, newest first.
//
// Registrations in every state are returned — in progress, completed, abandoned, and expired — so customer service can follow up on the ones that stalled before completing; narrow them with `status`. The search term matches the session ID and the customer name or number the buyer entered.
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
