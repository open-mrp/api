package constants

// MessageKind classifies what a message row represents.
type MessageKind string

const (
	// MessageKindChat is a user-authored chat message.
	MessageKindChat MessageKind = "chat"
	// MessageKindSystemEvent is a system-generated event message.
	MessageKindSystemEvent MessageKind = "system_event"
	// MessageKindAgent is a message authored by an AI agent participant.
	MessageKindAgent MessageKind = "agent"
	// MessageKindScheduled is a message materialized from a scheduled send.
	MessageKindScheduled MessageKind = "scheduled"
	// MessageKindAlert is a system/producer alert rendered as a message.
	MessageKindAlert MessageKind = "alert"
	// MessageKindEmail is an inbound email materialized into a conversation by the email bridge.
	MessageKindEmail MessageKind = "email"
)

func (k MessageKind) IsValid() bool {
	switch k {
	case MessageKindChat, MessageKindSystemEvent, MessageKindAgent, MessageKindScheduled, MessageKindAlert, MessageKindEmail:
		return true
	default:
		return false
	}
}

func (k MessageKind) EnumValues() []string {
	return []string{
		string(MessageKindChat),
		string(MessageKindSystemEvent),
		string(MessageKindAgent),
		string(MessageKindScheduled),
		string(MessageKindAlert),
		string(MessageKindEmail),
	}
}

func (k *MessageKind) StringPtr() *string {
	if k == nil {
		return nil
	}
	v := string(*k)
	return &v
}
