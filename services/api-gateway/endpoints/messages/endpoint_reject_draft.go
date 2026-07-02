package messageep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to discard an open customer-reply draft without sending it.
type RejectDraftRequest struct {
	// Message ID (the draft).
	MessageID string `path:"id" validate:"required"`
}

// Discards an open customer-reply draft without sending it (status becomes `rejected`).
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
