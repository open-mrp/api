package messaginggroupep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to rename a reusable roster.
type UpdateMessagingGroupRequest struct {
	// Messaging group ID.
	GroupID string `path:"id" validate:"required"`
	// The roster's new display name.
	Name string `json:"name" validate:"required"`
}

var sampleUpdateMessagingGroupRequest = &UpdateMessagingGroupRequest{
	GroupID: apiresource.SampleMessagingGroupID,
	Name:    "Operations Team",
}

func (*UpdateMessagingGroupRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateMessagingGroupRequest)
}

// Renames a reusable roster.
//
// Members are managed through the add-member and remove-member endpoints, not here.
type UpdateMessagingGroupEndpoint struct{}

func (e *UpdateMessagingGroupEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateMessagingGroupRequest, *apiresource.MessagingGroup] {
	return (&apiendpoint.APIEndpoint[*UpdateMessagingGroupRequest, *apiresource.MessagingGroup]{
		Title:               "Update Messaging Group",
		Method:              http.MethodPatch,
		ContentType:         "application/json",
		Route:               "/v1/messaging/groups/{id}",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		Preview:             true,
		ObjectType:          constants.ObjectTypeMessagingGroup,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionUpdate}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateMessagingGroupRequest) (*apiresource.MessagingGroup, *apierror.APIError) {
			return svc.(MessagingGroupSvc).UpdateMessagingGroup
		},
	})
}
