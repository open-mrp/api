package messaginggroupep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list the account's reusable rosters.
type ListMessagingGroupsRequest struct{}

// Lists the reusable rosters in the caller's account (most-recently-updated first).
type ListMessagingGroupsEndpoint struct{}

func (e *ListMessagingGroupsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListMessagingGroupsRequest, *apiresource.List[apiresource.MessagingGroup]] {
	return (&apiendpoint.APIEndpoint[*ListMessagingGroupsRequest, *apiresource.List[apiresource.MessagingGroup]]{
		Title:               "List Messaging Groups",
		Method:              http.MethodGet,
		Route:               "/v1/messaging/groups",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		Preview:             true,
		ObjectType:          constants.ObjectTypeMessagingGroup,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionRead}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListMessagingGroupsRequest) (*apiresource.List[apiresource.MessagingGroup], *apierror.APIError) {
			return svc.(MessagingGroupSvc).ListMessagingGroups
		},
	})
}
