package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleMessageAttachmentID = "mgah_01h9z8q1w2e3r4t5y6u7mgah"

// A file, image, link, or resource attached to a message.
type MessageAttachment struct {
	// Attachment ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=message_attachment"`
	// The kind of attachment, which determines how it is stored and which of the fields below are populated.
	//
	// - `file`: an uploaded non-image file.
	// - `image`: an uploaded image.
	// - `link`: an external URL reference, with no stored file.
	// - `resource`: a reference to an in-app resource, such as an order.
	Kind constants.MessageAttachmentKind `json:"kind" validate:"required"`
	// The original filename for uploaded attachments.
	//
	// `null` for link/resource attachments.
	Filename *string `json:"filename"`
	// The MIME content type for uploaded attachments.
	//
	// `null` for link/resource attachments.
	ContentType *string `json:"content_type"`
	// The size in bytes for uploaded attachments.
	//
	// `null` when unknown or for link/resource attachments.
	SizeBytes *int64 `json:"size_bytes"`
	// A time-limited download URL for uploaded (file/image) attachments, or the link URL.
	//
	// `null` for resource attachments.
	URL *string `json:"url"`
	// The linked in-app resource for `resource` attachments.
	//
	// `null` for file/image/link attachments.
	Resource *Entity `json:"resource" expandable:"true"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
}

var SampleMessageAttachment = &MessageAttachment{
	ID:          SampleMessageAttachmentID,
	Object:      constants.ObjectTypeMessageAttachment,
	Kind:        constants.MessageAttachmentKindImage,
	Filename:    new("diagram.png"),
	ContentType: new("image/png"),
	SizeBytes:   new(int64(48213)),
	URL:         new("https://chat-bucket.s3.amazonaws.com/chat/ac/cv/mgah/diagram.png?X-Amz-Signature=..."),
	CreatedAt:   timeutil.TimestampToTime(sampleCreatedAtTimestamp),
}

func (*MessageAttachment) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleMessageAttachment)
}
