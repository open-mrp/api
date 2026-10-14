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

// Request to remove a member from a reusable roster.
type RemoveMessagingGroupMemberRequest struct {
	// Messaging group ID.
	GroupID string `path:"id" validate:"required"`
	// The membership ID to remove (from the roster's `members`).
	MemberID string `path:"member_id" validate:"required"`
}

// Removes a member from a messaging group.
//
// Only conversations started from the group afterwards are affected; the member stays in every conversation already started from it.
type RemoveMessagingGroupMemberEndpoint struct{}

func (e *RemoveMessagingGroupMemberEndpoint) Materialize() *apiendpoint.APIEndpoint[*RemoveMessagingGroupMemberRequest, *object.Deleted] {
	return (&apiendpoint.APIEndpoint[*RemoveMessagingGroupMemberRequest, *object.Deleted]{
		Title:             "Remove Messaging Group Member",
		Method:            http.MethodDelete,
		Route:             "/v1/messaging/groups/{id}/members/{member_id}",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		ObjectType:        constants.ObjectTypeMessagingGroupMember,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionUpdate}}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *RemoveMessagingGroupMemberRequest) (*object.Deleted, *apierror.APIError) {
			return apiresource.Deleted(svc.(MessagingGroupSvc).RemoveMessagingGroupMember, constants.ObjectTypeMessagingGroupMember, "member_id")
		},
	})
}
