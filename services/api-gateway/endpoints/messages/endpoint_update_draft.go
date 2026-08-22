package messageep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/field"
)

// Request to edit a still-open customer-reply draft message.
type UpdateDraftRequest struct {
	// The id of the draft to edit.
	MessageID string `path:"id" validate:"required"`
	// The revised reply body, replacing what the draft said before.
	Body string `json:"body" validate:"required"`
	// The revised subject line for a draft that will be sent by email.
	//
	// Leaving it out keeps the draft's current subject.
	Subject field.Optional[string] `json:"subject,omitzero"`
}

var sampleUpdateDraftSubject = "Re: Order #1042"

var sampleUpdateDraftRequest = &UpdateDraftRequest{
	MessageID: apiresource.SampleMessageID,
	Body:      "Hi Joe — good news, your order ships tomorrow.",
	Subject:   field.Some(sampleUpdateDraftSubject),
}

func (*UpdateDraftRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateDraftRequest)
}

// Revises a reply draft before it is sent to the customer.
//
// Only a draft that is still awaiting approval can be edited; once it has been approved, rejected, or superseded the request fails. Nothing reaches the customer until the draft is approved.
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
