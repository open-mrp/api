package messaginggroupep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to create a reusable roster.
type CreateMessagingGroupRequest struct {
	// The roster's display name.
	Name string `json:"name" validate:"required"`
	// The account users to include in the roster.
	MemberAccountUserIDs []string `json:"member_account_user_ids,omitzero"`
	// The agents to include in the roster.
	MemberAgentConfigIDs []string `json:"member_agent_config_ids,omitzero"`
}

var sampleCreateMessagingGroupRequest = &CreateMessagingGroupRequest{
	Name:                 "Operations Team",
	MemberAccountUserIDs: []string{apiresource.SampleAccountUserID},
}

func (*CreateMessagingGroupRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateMessagingGroupRequest)
}

// Creates a reusable roster of members (users and/or agents) that can seed many conversations.
type CreateMessagingGroupEndpoint struct{}

func (e *CreateMessagingGroupEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateMessagingGroupRequest, *apiresource.MessagingGroup] {
	return (&apiendpoint.APIEndpoint[*CreateMessagingGroupRequest, *apiresource.MessagingGroup]{
		Title:               "Create Messaging Group",
		Method:              http.MethodPost,
		ContentType:         "application/json",
		Route:               "/v1/messaging/groups",
		SuccessStatusCode:   http.StatusCreated,
		Public:              true,
		Preview:             true,
		ObjectType:          constants.ObjectTypeMessagingGroup,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionCreate}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateMessagingGroupRequest) (*apiresource.MessagingGroup, *apierror.APIError) {
			return svc.(MessagingGroupSvc).CreateMessagingGroup
		},
		LocationFunc: func(resp *apiresource.MessagingGroup) string {
			return "/v1/messaging/groups/" + resp.ID
		},
	})
}
