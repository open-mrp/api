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
	// The id of the draft to approve.
	MessageID string `path:"id" validate:"required"`
	// A unique client-generated key for this approval, such as a UUID.
	ClientMessageID string `json:"client_message_id" validate:"required"`
}

var sampleApproveSendDraftRequest = &ApproveSendDraftRequest{
	MessageID:       apiresource.SampleMessageID,
	ClientMessageID: "client_msg_approve_7b1c",
}

func (*ApproveSendDraftRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleApproveSendDraftRequest)
}

// Approves a reply draft and sends it to the customer.
//
// The draft becomes the sent message rather than spawning a copy: it takes its place in the case timeline, and the customer sees it as coming from "Customer Service". A draft on the email channel goes out as a reply on the case's email thread; otherwise it appears in the customer's conversation. Sending also moves the case to waiting on the customer.
//
// Only the first approval of a draft sends it — approving one that is no longer open fails, so a concurrent double-approve cannot reach the customer twice. Customer accounts cannot approve drafts.
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
