package messageep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to discard an open customer-reply draft without sending it.
type RejectDraftRequest struct {
	// The id of the draft to reject.
	MessageID string `path:"id" validate:"required"`
}

// Discards a reply draft without sending it to the customer.
//
// The draft is kept as a rejected record for history and can no longer be edited or approved. Because the customer is still owed an answer, the case moves back to waiting on your team.
type RejectDraftEndpoint struct{}

func (e *RejectDraftEndpoint) Materialize() *apiendpoint.APIEndpoint[*RejectDraftRequest, *apiresource.Message] {
	return (&apiendpoint.APIEndpoint[*RejectDraftRequest, *apiresource.Message]{
		Title:               "Reject Reply Draft",
		Method:              http.MethodPost,
		ContentType:         "application/json",
		Route:               "/v1/messaging/messages/{id}/actions/reject",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		Preview:             true,
		ObjectType:          constants.ObjectTypeChatMessage,
		IncludeConfig:       messageIncludeConfig(),
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionUpdate}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *RejectDraftRequest) (*apiresource.Message, *apierror.APIError) {
			return svc.(MessageSvc).RejectDraft
		},
	})
}
