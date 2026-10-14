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
	apiexample "github.com/open-mrp/apikit/example"
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
	MemberAgentConfigIDs: []string{apiresource.SampleAgentDefinitionID},
}

func (*CreateMessagingGroupRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateMessagingGroupRequest)
}

// Creates a reusable roster of members (users and/or agents) that can seed many conversations.
//
// Every account user listed must exist; repeated ids are ignored. The caller is recorded as the creator but is not added to the roster automatically — include their own account user id to be a member.
type CreateMessagingGroupEndpoint struct{}

func (e *CreateMessagingGroupEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateMessagingGroupRequest, *apiresource.MessagingGroup] {
	return (&apiendpoint.APIEndpoint[*CreateMessagingGroupRequest, *apiresource.MessagingGroup]{
		Title:             "Create Messaging Group",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/messaging/groups",
		SuccessStatusCode: http.StatusCreated,
		Public:            true,
		ObjectType:        constants.ObjectTypeMessagingGroup,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionCreate}}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateMessagingGroupRequest) (*apiresource.MessagingGroup, *apierror.APIError) {
			return svc.(MessagingGroupSvc).CreateMessagingGroup
		},
		LocationFunc: func(resp *apiresource.MessagingGroup) string {
			return "/v1/messaging/groups/" + resp.ID
		},
	})
}
