package conversationep

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

// Request to link a business record to a conversation.
type AddConversationLinkRequest struct {
	// Conversation ID.
	ConversationID string `path:"id" validate:"required"`
	// The kind of business record to link.
	ResourceType constants.ObjectType `json:"resource_type" validate:"required"`
	// The id of the business record to link.
	ResourceID string `json:"resource_id" validate:"required"`
}

var sampleAddConversationLinkRequest = &AddConversationLinkRequest{
	ConversationID: apiresource.SampleConversationID,
	ResourceType:   constants.ObjectTypeSalesOrder,
	ResourceID:     apiresource.SampleSalesOrderID,
}

func (*AddConversationLinkRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleAddConversationLinkRequest)
}

// Links a business record to a conversation, in addition to whatever topic the conversation is anchored to.
//
// A conversation can link any number of records, and each linked record surfaces the conversation when conversations are listed for that record.
type AddConversationLinkEndpoint struct{}

func (e *AddConversationLinkEndpoint) Materialize() *apiendpoint.APIEndpoint[*AddConversationLinkRequest, *apiresource.ConversationLink] {
	return (&apiendpoint.APIEndpoint[*AddConversationLinkRequest, *apiresource.ConversationLink]{
		Title:               "Link Record",
		Method:              http.MethodPost,
		ContentType:         "application/json",
		Route:               "/v1/messaging/conversations/{id}/links",
		SuccessStatusCode:   http.StatusCreated,
		Public:              true,
		AgentTool:           true,
		Preview:             true,
		ObjectType:          constants.ObjectTypeConversationLink,
		IncludeConfig:       conversationLinkIncludeConfig(),
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionUpdate}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *AddConversationLinkRequest) (*apiresource.ConversationLink, *apierror.APIError) {
			return svc.(ConversationSvc).AddConversationLink
		},
	})
}
