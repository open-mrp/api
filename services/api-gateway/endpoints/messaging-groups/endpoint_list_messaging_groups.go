package messaginggroupep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

// Request to list the account's reusable rosters.
type ListMessagingGroupsRequest struct{}

// Lists the reusable rosters in the caller's account, each with its members.
//
// Rosters come back most-recently-updated first, and adding or removing a member counts as an update. The whole account's rosters are returned in one page.
type ListMessagingGroupsEndpoint struct{}

func (e *ListMessagingGroupsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListMessagingGroupsRequest, *apiresource.List[apiresource.MessagingGroup]] {
	return (&apiendpoint.APIEndpoint[*ListMessagingGroupsRequest, *apiresource.List[apiresource.MessagingGroup]]{
		Title:             "List Messaging Groups",
		Method:            http.MethodGet,
		Route:             "/v1/messaging/groups",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ObjectType:        constants.ObjectTypeMessagingGroup,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionRead}}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListMessagingGroupsRequest) (*apiresource.List[apiresource.MessagingGroup], *apierror.APIError) {
			return svc.(MessagingGroupSvc).ListMessagingGroups
		},
	})
}
