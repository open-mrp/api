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
	"github.com/open-mrp/apikit/object"
)

// Request to list the account's reusable rosters.
type ListMessagingGroupsRequest struct {
	object.ListRequest
}

// Returns a list of your messaging groups, each with its members.
//
// The groups are sorted by when they last changed, most recent first; adding or removing a member counts as a change.
type ListMessagingGroupsEndpoint struct{}

func (e *ListMessagingGroupsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListMessagingGroupsRequest, *apiresource.List[apiresource.MessagingGroup]] {
	return (&apiendpoint.APIEndpoint[*ListMessagingGroupsRequest, *apiresource.List[apiresource.MessagingGroup]]{
		Title:             "List Messaging Groups",
		Method:            http.MethodGet,
		Route:             "/v1/messaging/groups",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		ObjectType:        constants.ObjectTypeMessagingGroup,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionRead}}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListMessagingGroupsRequest) (*apiresource.List[apiresource.MessagingGroup], *apierror.APIError) {
			return svc.(MessagingGroupSvc).ListMessagingGroups
		},
	})
}
