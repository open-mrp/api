package constants

import "strings"

// MessageAttachmentKind classifies a message attachment. It is an enum so new kinds can be added without a breaking change to the API.
type MessageAttachmentKind string

const (
	// MessageAttachmentKindFile is an uploaded non-image file (stored in the chat bucket).
	MessageAttachmentKindFile MessageAttachmentKind = "file"
	// MessageAttachmentKindImage is an uploaded image (stored in the chat bucket).
	MessageAttachmentKindImage MessageAttachmentKind = "image"
	// MessageAttachmentKindLink is an external URL reference (no stored object).
	MessageAttachmentKindLink MessageAttachmentKind = "link"
	// MessageAttachmentKindResource is a typed in-app resource reference (e.g. an order).
	MessageAttachmentKindResource MessageAttachmentKind = "resource"
)

func (k MessageAttachmentKind) IsValid() bool {
	switch k {
	case MessageAttachmentKindFile, MessageAttachmentKindImage, MessageAttachmentKindLink, MessageAttachmentKindResource:
		return true
	default:
		return false
	}
}

func (k MessageAttachmentKind) EnumValues() []string {
	return []string{
		string(MessageAttachmentKindFile),
		string(MessageAttachmentKindImage),
		string(MessageAttachmentKindLink),
		string(MessageAttachmentKindResource),
	}
}

func (k *MessageAttachmentKind) StringPtr() *string {
	if k == nil {
		return nil
	}
	s := string(*k)
	return &s
}

// IsUploaded reports whether the kind is backed by a stored object in the chat bucket.
func (k MessageAttachmentKind) IsUploaded() bool {
	return k == MessageAttachmentKindFile || k == MessageAttachmentKindImage
}

// UploadedAttachmentKindForContentType picks file vs image for a staged upload from its MIME type.
func UploadedAttachmentKindForContentType(contentType string) MessageAttachmentKind {
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(contentType)), "image/") {
		return MessageAttachmentKindImage
	}
	return MessageAttachmentKindFile
}
