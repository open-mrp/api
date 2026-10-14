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

// Request to delete a reusable roster.
type DeleteMessagingGroupRequest struct {
	// Messaging group ID.
	GroupID string `path:"id" validate:"required"`
}

// Deletes a reusable roster.
//
// Conversations already started from it are unaffected (their members were snapshotted); they simply lose the roster reference.
type DeleteMessagingGroupEndpoint struct{}

func (e *DeleteMessagingGroupEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteMessagingGroupRequest, *object.Deleted] {
	return (&apiendpoint.APIEndpoint[*DeleteMessagingGroupRequest, *object.Deleted]{
		Title:             "Delete Messaging Group",
		Method:            http.MethodDelete,
		Route:             "/v1/messaging/groups/{id}",
		SuccessStatusCode: http.StatusOK,
		ObjectType:        constants.ObjectTypeMessagingGroup,
		Public:            true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionDelete}}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteMessagingGroupRequest) (*object.Deleted, *apierror.APIError) {
			return apiresource.Deleted(svc.(MessagingGroupSvc).DeleteMessagingGroup, constants.ObjectTypeMessagingGroup, "id")
		},
	})
}
