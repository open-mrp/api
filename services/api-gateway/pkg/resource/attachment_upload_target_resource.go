package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

// A presigned target for uploading a chat attachment directly to object storage.
//
// The client PUTs the file to `upload_url`, then references `s3_key` when sending the message. Request `?include=attachment` to expand the pre-allocated attachment metadata.
type AttachmentUploadTarget struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=attachment_upload_target"`
	// Pre-allocated attachment metadata for the staged upload.
	Attachment *MessageAttachment `json:"attachment" expandable:"true"`
	// The presigned URL to PUT the file to.
	UploadURL string `json:"upload_url" validate:"required"`
	// The object-storage key to echo back when sending the message.
	S3Key string `json:"s3_key" validate:"required"`
	// When the upload URL expires.
	ExpiresAt time.Time `json:"expires_at" validate:"required"`
}

var SampleAttachmentUploadTarget = &AttachmentUploadTarget{
	Object:     constants.ObjectTypeAttachmentUploadTarget,
	Attachment: SampleMessageAttachment,
	UploadURL:  "https://chat-bucket.s3.amazonaws.com/staged/ac/cv/mgah/diagram.png?X-Amz-Signature=...",
	S3Key:      "staged/ac_01h9z8q1w2e3r4t5y6u7i8o9/cv_01h9z8q1w2e3r4t5y6u7i8cv/mgah_01h9z8q1w2e3r4t5y6u7mgah/diagram.png",
	ExpiresAt:  timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*AttachmentUploadTarget) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleAttachmentUploadTarget)
}
