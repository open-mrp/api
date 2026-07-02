package messaginggroupep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	apierror "github.com/augno/api/shared/errors"
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

func (e *DeleteMessagingGroupEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteMessagingGroupRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*DeleteMessagingGroupRequest, *apiresource.EmptyResource]{
		Title:               "Delete Messaging Group",
		Method:              http.MethodDelete,
		Route:               "/v1/messaging/groups/{id}",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		Preview:             true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionDelete}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteMessagingGroupRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(MessagingGroupSvc).DeleteMessagingGroup
		},
	})
}
