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

// Request to remove a member from a reusable roster.
type RemoveMessagingGroupMemberRequest struct {
	// Messaging group ID.
	GroupID string `path:"id" validate:"required"`
	// The membership ID to remove (from the roster's `members`).
	MemberID string `path:"member_id" validate:"required"`
}

// Removes a member from a reusable roster and returns the updated roster.
//
// Only conversations started from the roster afterwards are affected; the member stays in every conversation that was already seeded from it.
type RemoveMessagingGroupMemberEndpoint struct{}

func (e *RemoveMessagingGroupMemberEndpoint) Materialize() *apiendpoint.APIEndpoint[*RemoveMessagingGroupMemberRequest, *apiresource.MessagingGroup] {
	return (&apiendpoint.APIEndpoint[*RemoveMessagingGroupMemberRequest, *apiresource.MessagingGroup]{
		Title:               "Remove Messaging Group Member",
		Method:              http.MethodDelete,
		Route:               "/v1/messaging/groups/{id}/members/{member_id}",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		Preview:             true,
		ObjectType:          constants.ObjectTypeMessagingGroup,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionUpdate}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *RemoveMessagingGroupMemberRequest) (*apiresource.MessagingGroup, *apierror.APIError) {
			return svc.(MessagingGroupSvc).RemoveMessagingGroupMember
		},
	})
}
