package attachmentep

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

// Request to mint a presigned upload target for a chat attachment.
type CreateAttachmentUploadURLRequest struct {
	// Conversation ID the attachment will be sent in.
	//
	// The upload can only be attached to a message in this conversation.
	ConversationID string `path:"id" validate:"required"`
	// The original filename of the file to upload.
	Filename string `json:"filename" validate:"required"`
	// The MIME content type of the file.
	//
	// The file must then be uploaded with this same content type, or object storage rejects it. It also decides how the attachment preview returned here is classified: `image/…` becomes an inline image, anything else a file.
	ContentType field.Optional[string] `json:"content_type,omitzero"`
}

var sampleCreateAttachmentUploadURLRequest = &CreateAttachmentUploadURLRequest{
	ConversationID: apiresource.SampleConversationID,
	Filename:       "diagram.png",
	ContentType:    field.Some("image/png"),
}

func (*CreateAttachmentUploadURLRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateAttachmentUploadURLRequest)
}

// Creates a short-lived URL for uploading a chat attachment straight to object storage.
//
// Upload the file to the returned URL, then send a message in the same conversation carrying the returned key as an attachment — the file only becomes part of the conversation at that point, and an upload that is never sent is discarded automatically. You must be an active participant of the conversation to stage an upload for it.
type CreateAttachmentUploadURLEndpoint struct{}

func (e *CreateAttachmentUploadURLEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateAttachmentUploadURLRequest, *apiresource.AttachmentUploadTarget] {
	return (&apiendpoint.APIEndpoint[*CreateAttachmentUploadURLRequest, *apiresource.AttachmentUploadTarget]{
		Title:               "Create Attachment Upload URL",
		Method:              http.MethodPost,
		ContentType:         "application/json",
		Route:               "/v1/messaging/conversations/{id}/attachments/actions/upload-url",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		Preview:             true,
		ObjectType:          constants.ObjectTypeAttachmentUploadTarget,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionCreate}},
		IncludeConfig:       attachmentUploadTargetIncludeConfig(),
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateAttachmentUploadURLRequest) (*apiresource.AttachmentUploadTarget, *apierror.APIError) {
			return svc.(AttachmentSvc).CreateAttachmentUploadURL
		},
	})
}
