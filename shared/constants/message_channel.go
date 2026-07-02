package constants

// MessageChannel is how a message was delivered (or, for a draft, how it will be delivered on approve): an in-conversation chat message or email via the bridged inbox.
type MessageChannel string

const (
	// MessageChannelMessage delivers as an in-conversation chat message (the customer portal timeline for external cases).
	MessageChannelMessage MessageChannel = "message"
	// MessageChannelEmail delivers as email through the conversation's bridged inbox.
	MessageChannelEmail MessageChannel = "email"
)

func (c MessageChannel) IsValid() bool {
	switch c {
	case MessageChannelMessage, MessageChannelEmail:
		return true
	default:
		return false
	}
}

func (c MessageChannel) EnumValues() []string {
	return []string{string(MessageChannelMessage), string(MessageChannelEmail)}
}

func (c *MessageChannel) StringPtr() *string {
	if c == nil {
		return nil
	}
	s := string(*c)
	return &s
}

func MessageChannelPtr(c MessageChannel) *string {
	s := string(c)
	return &s
}

// ResolveMessageChannel normalizes a stored channel value (including legacy "portal") and infers email from kind when absent.
func ResolveMessageChannel(stored *string, kind string) MessageChannel {
	if stored != nil && *stored != "" {
		switch MessageChannel(*stored) {
		case MessageChannelEmail:
			return MessageChannelEmail
		case MessageChannelMessage, MessageChannel("portal"):
			return MessageChannelMessage
		}
	}
	if kind == string(MessageKindEmail) {
		return MessageChannelEmail
	}
	return MessageChannelMessage
}
