package conversationep

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

// Request to place a conversation under legal hold or release it.
type SetLegalHoldRequest struct {
	// Conversation ID.
	ConversationID string `path:"id" validate:"required"`
	// Whether to place the conversation under legal hold or release it.
	//
	// - `held`: the conversation is preserved — exempt from automatic retention purging and from redaction.
	// - `released`: normal retention and redaction apply again.
	LegalHold constants.LegalHoldStatus `json:"legal_hold" validate:"required"`
}

var sampleSetLegalHoldRequest = &SetLegalHoldRequest{
	ConversationID: apiresource.SampleConversationID,
	LegalHold:      constants.LegalHoldStatusHeld,
}

func (*SetLegalHoldRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleSetLegalHoldRequest)
}

// Places a conversation under legal hold or releases it.
//
// Holding it exempts the conversation from automatic retention purging, and any attempt to redact it is refused until the hold is released.
type SetLegalHoldEndpoint struct{}

func (e *SetLegalHoldEndpoint) Materialize() *apiendpoint.APIEndpoint[*SetLegalHoldRequest, *apiresource.Conversation] {
	return (&apiendpoint.APIEndpoint[*SetLegalHoldRequest, *apiresource.Conversation]{
		Title:               "Set Legal Hold",
		Method:              http.MethodPost,
		ContentType:         "application/json",
		Route:               "/v1/messaging/conversations/{id}/actions/set-legal-hold",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           false,
		Preview:             true,
		ObjectType:          constants.ObjectTypeConversation,
		IncludeConfig:       conversationIncludeConfig(),
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionUpdate}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *SetLegalHoldRequest) (*apiresource.Conversation, *apierror.APIError) {
			return svc.(ConversationSvc).SetLegalHold
		},
	})
}
