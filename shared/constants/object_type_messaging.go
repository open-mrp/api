package constants

import "github.com/open-mrp/apikit/object"

// Object types of the `related` groups on messaging resources.
const (
	// ObjectTypeConversationRelated indicates that the object is the records a conversation is about or picks up from.
	ObjectTypeConversationRelated ObjectType = "conversation_related"
	// ObjectTypeChatMessageRelated indicates that the object is the record a message links to.
	ObjectTypeChatMessageRelated ObjectType = "chat_message_related"
	// ObjectTypeAnnouncementRelated indicates that the object is the record an announcement is about.
	ObjectTypeAnnouncementRelated ObjectType = "announcement_related"
	// ObjectTypeNotificationRelated indicates that the object is the record a notification is about.
	ObjectTypeNotificationRelated ObjectType = "notification_related"
)

func init() {
	object.Register(ObjectTypeConversationRelated, ObjectTypeChatMessageRelated, ObjectTypeAnnouncementRelated, ObjectTypeNotificationRelated)
}
