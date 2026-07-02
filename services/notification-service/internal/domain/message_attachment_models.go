package domain

import "time"

// MessageAttachment is a file/image/link/resource attached to a message. Uploaded kinds (file/image)
// reference an object in the chat bucket by S3Key; URL is a presigned GET resolved at read time.
type MessageAttachment struct {
	ID           string
	MessageID    string
	AccountID    string
	Kind         string
	S3Key        *string
	URL          *string
	Filename     *string
	ContentType  *string
	SizeBytes    *int64
	ResourceType *string
	ResourceID   *string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// AttachmentInput is one attachment supplied on a send-message request. For uploaded kinds the client echoes the S3Key it received from the upload-url endpoint; the server verifies the object exists (and that the key is scoped to the conversation) before persisting.
type AttachmentInput struct {
	Kind         string
	S3Key        *string
	Filename     *string
	ContentType  *string
	SizeBytes    *int64
	URL          *string
	ResourceType *string
	ResourceID   *string
}

// AttachmentUploadTarget is the result of requesting an upload URL: a presigned PUT plus a pre-allocated attachment preview and the staging key the client echoes back when sending the message.
type AttachmentUploadTarget struct {
	Attachment *MessageAttachment
	UploadURL  string
	S3Key      string
	ExpiresAt  time.Time
}
