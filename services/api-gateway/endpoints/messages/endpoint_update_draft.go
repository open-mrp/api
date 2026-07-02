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
	"github.com/augno/api/shared/field"
)

// Request to edit a still-open customer-reply draft message.
type UpdateDraftRequest struct {
	// Message ID.
	MessageID string `path:"id" validate:"required"`
	// The revised reply body.
	Body string `json:"body" validate:"required"`
	// The revised email subject, for the email channel.
	Subject field.Optional[string] `json:"subject,omitzero"`
}

var sampleUpdateDraftRequest = &UpdateDraftRequest{
	MessageID: apiresource.SampleMessageID,
	Body:      "Hi Joe — good news, your order ships tomorrow.",
}

func (*UpdateDraftRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateDraftRequest)
}

// Edits a still-open customer-reply draft.
type UpdateDraftEndpoint struct{}

func (e *UpdateDraftEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateDraftRequest, *apiresource.Message] {
	return (&apiendpoint.APIEndpoint[*UpdateDraftRequest, *apiresource.Message]{
		Title:               "Update Reply Draft",
		Method:              http.MethodPatch,
		ContentType:         "application/json",
		Route:               "/v1/messaging/messages/{id}",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		Preview:             true,
		ObjectType:          constants.ObjectTypeChatMessage,
		IncludeConfig:       messageIncludeConfig(),
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionUpdate}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateDraftRequest) (*apiresource.Message, *apierror.APIError) {
			return svc.(MessageSvc).UpdateDraft
		},
	})
}
