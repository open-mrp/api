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
	ConversationID string `path:"id" validate:"required"`
	// The original filename of the file to upload.
	Filename string `json:"filename" validate:"required"`
	// The MIME content type of the file (sent as the Content-Type on the upload).
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

// Mints a presigned URL for uploading a chat attachment directly to object storage.
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
