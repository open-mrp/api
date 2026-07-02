package constants

// MessageVisibility is the audience of a single message inside a conversation. It is the central safety primitive for external (audience=customer) conversations: an internal note is never serialized into a customer payload, while an external message is part of the official customer communication history. Internal conversations force Internal. System is for events visible to both parties (linked-record added, draft generated, status changed).
type MessageVisibility string

const (
	// MessageVisibilityInternal is a team-only message (internal note / private discussion).
	MessageVisibilityInternal MessageVisibility = "internal"
	// MessageVisibilityExternal is a message sent to or received from an external party (e.g. the customer on a support case).
	MessageVisibilityExternal MessageVisibility = "external"
	// MessageVisibilitySystem is a system/event message shown to both the team and the customer.
	MessageVisibilitySystem MessageVisibility = "system"
)

func (v MessageVisibility) IsValid() bool {
	switch v {
	case MessageVisibilityInternal, MessageVisibilityExternal, MessageVisibilitySystem:
		return true
	default:
		return false
	}
}

func (v MessageVisibility) EnumValues() []string {
	return []string{string(MessageVisibilityInternal), string(MessageVisibilityExternal), string(MessageVisibilitySystem)}
}

func (v *MessageVisibility) StringPtr() *string {
	if v == nil {
		return nil
	}
	s := string(*v)
	return &s
}
