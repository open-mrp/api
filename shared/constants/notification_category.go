package constants

// NotificationCategory classifies an in-app notification. The set is intentionally open-ended (producers may introduce new categories), so notification fields use
// `validate:"required"` rather than a strict enum tag; the named constants below cover the common, first-party categories.
type NotificationCategory string

const (
	// NotificationCategoryChatMessage is a new chat message in a conversation.
	NotificationCategoryChatMessage NotificationCategory = "chat.message"
	// NotificationCategoryChatMention is a direct @mention (pierces mute).
	NotificationCategoryChatMention NotificationCategory = "chat.mention"
	// NotificationCategoryChatAdded indicates the user was added to a conversation.
	NotificationCategoryChatAdded NotificationCategory = "chat.added"
	// NotificationCategoryOrderUpdated indicates a change to an order the user is involved with.
	NotificationCategoryOrderUpdated NotificationCategory = "order.updated"
	// NotificationCategoryAgentRunCompleted indicates an agent run the user triggered finished.
	NotificationCategoryAgentRunCompleted NotificationCategory = "agent.run_completed"
	// NotificationCategoryAgentAlert is an alert an agent raised during a run.
	NotificationCategoryAgentAlert NotificationCategory = "agent.alert"
	// NotificationCategorySystemBroadcast is a targeted system message to a user.
	NotificationCategorySystemBroadcast NotificationCategory = "system.broadcast"
)

func (c NotificationCategory) IsValid() bool {
	switch c {
	case NotificationCategoryChatMessage,
		NotificationCategoryChatMention,
		NotificationCategoryChatAdded,
		NotificationCategoryOrderUpdated,
		NotificationCategoryAgentRunCompleted,
		NotificationCategoryAgentAlert,
		NotificationCategorySystemBroadcast:
		return true
	default:
		return false
	}
}

func (c NotificationCategory) EnumValues() []string {
	return []string{
		string(NotificationCategoryChatMessage),
		string(NotificationCategoryChatMention),
		string(NotificationCategoryChatAdded),
		string(NotificationCategoryOrderUpdated),
		string(NotificationCategoryAgentRunCompleted),
		string(NotificationCategoryAgentAlert),
		string(NotificationCategorySystemBroadcast),
	}
}

func (c *NotificationCategory) StringPtr() *string {
	if c == nil {
		return nil
	}
	v := string(*c)
	return &v
}
