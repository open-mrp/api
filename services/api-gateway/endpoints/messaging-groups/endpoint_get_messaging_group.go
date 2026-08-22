package messaginggroupep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to retrieve one reusable roster.
type GetMessagingGroupRequest struct {
	// Messaging group ID.
	GroupID string `path:"id" validate:"required"`
}

// Retrieves a reusable roster together with its current members.
type GetMessagingGroupEndpoint struct{}

func (e *GetMessagingGroupEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetMessagingGroupRequest, *apiresource.MessagingGroup] {
	return (&apiendpoint.APIEndpoint[*GetMessagingGroupRequest, *apiresource.MessagingGroup]{
		Title:               "Retrieve Messaging Group",
		Method:              http.MethodGet,
		Route:               "/v1/messaging/groups/{id}",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		Preview:             true,
		ObjectType:          constants.ObjectTypeMessagingGroup,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionRead}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetMessagingGroupRequest) (*apiresource.MessagingGroup, *apierror.APIError) {
			return svc.(MessagingGroupSvc).GetMessagingGroup
		},
	})
}
