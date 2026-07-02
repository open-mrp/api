package messageep

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

// Request to approve a customer-reply draft and send it to the customer.
type ApproveSendDraftRequest struct {
	// Message ID (the draft).
	MessageID string `path:"id" validate:"required"`
	// Client-supplied dedupe key for the resulting customer-visible message.
	ClientMessageID string `json:"client_message_id" validate:"required"`
}

var sampleApproveSendDraftRequest = &ApproveSendDraftRequest{
	MessageID:       apiresource.SampleMessageID,
	ClientMessageID: "client_msg_approve_7b1c",
}

func (*ApproveSendDraftRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleApproveSendDraftRequest)
}

// Approves a customer-reply draft and sends it to the customer, promoting the draft to a sent customer-visible message in place.
//
// Idempotent on `client_message_id`.
type ApproveSendDraftEndpoint struct{}

func (e *ApproveSendDraftEndpoint) Materialize() *apiendpoint.APIEndpoint[*ApproveSendDraftRequest, *apiresource.Message] {
	return (&apiendpoint.APIEndpoint[*ApproveSendDraftRequest, *apiresource.Message]{
		Title:               "Approve And Send Reply Draft",
		Method:              http.MethodPost,
		ContentType:         "application/json",
		Route:               "/v1/messaging/messages/{id}/actions/approve-send",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		Preview:             true,
		ObjectType:          constants.ObjectTypeChatMessage,
		IncludeConfig:       messageIncludeConfig(),
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionUpdate}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ApproveSendDraftRequest) (*apiresource.Message, *apierror.APIError) {
			return svc.(MessageSvc).ApproveSendDraft
		},
	})
}
