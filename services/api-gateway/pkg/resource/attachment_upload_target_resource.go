package apiresource

import (
	"time"

	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/timeutil"
)

// A presigned target for uploading a chat attachment directly to object storage.
//
// PUT the file to `upload_url`, then send a message carrying an attachment whose `s3_key` is the key returned here. An upload that is never sent with a message is discarded automatically, so abandoning a target costs nothing.
type AttachmentUploadTarget struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=attachment_upload_target"`
	// A preview of the attachment the file becomes once it is sent with a message.
	Attachment *MessageAttachment `json:"attachment" expandable:"true"`
	// The presigned URL to PUT the file to.
	//
	// Send the file with the same content type used to mint the target, or the upload is rejected.
	UploadURL string `json:"upload_url" validate:"required"`
	// The object-storage key identifying the uploaded file.
	//
	// Pass it back as an attachment's `s3_key` when sending a message. It is bound to the conversation it was minted for and cannot be attached in another one.
	S3Key string `json:"s3_key" validate:"required"`
	// When the upload URL stops working.
	//
	// Targets are short-lived (about fifteen minutes); request a new one if the upload has not finished by then.
	ExpiresAt time.Time `json:"expires_at" validate:"required"`
}

var SampleAttachmentUploadTarget = &AttachmentUploadTarget{
	Object:     constants.ObjectTypeAttachmentUploadTarget,
	Attachment: SampleMessageAttachment,
	UploadURL:  "https://chat-bucket.s3.amazonaws.com/staged/ac/cv/mgah/diagram.png?X-Amz-Signature=...",
	S3Key:      "staged/ac_01h9z8q1w2e3r4t5y6u7i8o9/cv_w35z4ck68yq7/mgah_v17axle2mcff/diagram.png",
	ExpiresAt:  timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*AttachmentUploadTarget) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleAttachmentUploadTarget)
}
