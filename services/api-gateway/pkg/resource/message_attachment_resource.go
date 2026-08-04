package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleMessageAttachmentID = "mgah_v17axle2mcff"

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
	// The filename the attachment was uploaded under.
	//
	// Carried only by `file` and `image` attachments.
	Filename *string `json:"filename"`
	// The MIME type of the uploaded content.
	//
	// Carried only by `file` and `image` attachments.
	ContentType *string `json:"content_type"`
	// The size of the uploaded content in bytes.
	//
	// Carried only by `file` and `image` attachments, and only when the sender supplied it with the message.
	SizeBytes *int64 `json:"size_bytes"`
	// Where to fetch the attachment: a signed download URL for `file` and `image` attachments, or the target address for `link` attachments.
	//
	// Download URLs are signed for one hour and regenerated each time the message is read, so follow the URL promptly instead of persisting it. `resource` attachments have no URL — use `resource` to resolve them.
	URL *string `json:"url"`
	// The in-app record a `resource` attachment points to, such as a sales order.
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
