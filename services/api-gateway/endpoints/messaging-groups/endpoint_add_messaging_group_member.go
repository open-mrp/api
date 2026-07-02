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
	"github.com/augno/api/shared/field"
)

// Request to add a member to a reusable roster.
type AddMessagingGroupMemberRequest struct {
	// Messaging group ID.
	GroupID string `path:"id" validate:"required"`
	// The kind of member being added.
	MemberType constants.MessagingGroupMemberType `json:"member_type" validate:"required"`
	// The account user to add (required when `member_type` is `user`).
	AccountUserID field.Optional[string] `json:"account_user_id,omitzero"`
	// The agent to add (required when `member_type` is `agent`).
	AgentConfigID field.Optional[string] `json:"agent_config_id,omitzero"`
}

var sampleAddMessagingGroupMemberRequest = &AddMessagingGroupMemberRequest{
	GroupID:       apiresource.SampleMessagingGroupID,
	MemberType:    constants.MessagingGroupMemberTypeUser,
	AccountUserID: field.Some(apiresource.SampleAccountUserID),
}

func (*AddMessagingGroupMemberRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleAddMessagingGroupMemberRequest)
}

// Adds a member (a user or an agent) to a reusable roster.
type AddMessagingGroupMemberEndpoint struct{}

func (e *AddMessagingGroupMemberEndpoint) Materialize() *apiendpoint.APIEndpoint[*AddMessagingGroupMemberRequest, *apiresource.MessagingGroup] {
	return (&apiendpoint.APIEndpoint[*AddMessagingGroupMemberRequest, *apiresource.MessagingGroup]{
		Title:               "Add Messaging Group Member",
		Method:              http.MethodPost,
		ContentType:         "application/json",
		Route:               "/v1/messaging/groups/{id}/members",
		SuccessStatusCode:   http.StatusCreated,
		Public:              true,
		Preview:             true,
		ObjectType:          constants.ObjectTypeMessagingGroup,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionUpdate}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *AddMessagingGroupMemberRequest) (*apiresource.MessagingGroup, *apierror.APIError) {
			return svc.(MessagingGroupSvc).AddMessagingGroupMember
		},
	})
}
