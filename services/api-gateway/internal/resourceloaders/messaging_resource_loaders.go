package resourceloaders

import (
	"context"

	apierror "github.com/augno/api/shared/errors"
)

// The notification, announcement, message_report, and messaging_block resources are only ever produced as top-level roots of their own endpoints — never referenced by id as an include target — so these loaders are never invoked. They exist solely to satisfy the resourcekit Definition's required non-nil Load.

// LoadNotifications is the never-invoked loader for the notification resource.
func LoadNotifications(_ context.Context, _ []string) (map[string]any, *apierror.APIError) {
	return nil, nil
}

// LoadAnnouncements is the never-invoked loader for the announcement resource.
func LoadAnnouncements(_ context.Context, _ []string) (map[string]any, *apierror.APIError) {
	return nil, nil
}

// LoadMessageReports is the never-invoked loader for the message_report resource.
func LoadMessageReports(_ context.Context, _ []string) (map[string]any, *apierror.APIError) {
	return nil, nil
}

// LoadMessagingBlocks is the never-invoked loader for the messaging_block resource.
func LoadMessagingBlocks(_ context.Context, _ []string) (map[string]any, *apierror.APIError) {
	return nil, nil
}

// LoadMessageAttachments is the never-invoked loader for the message_attachment resource. Attachments arrive inline on their parent message and are resolved by traversal (message ?include=attachments), never fetched by id, so this loader is never invoked.
func LoadMessageAttachments(_ context.Context, _ []string) (map[string]any, *apierror.APIError) {
	return nil, nil
}

// LoadConversationLinks is the never-invoked loader for the conversation_link resource. Links are only ever produced as the roots of the conversation-links endpoints, never fetched by id as an include target, so this loader is never invoked.
func LoadConversationLinks(_ context.Context, _ []string) (map[string]any, *apierror.APIError) {
	return nil, nil
}
