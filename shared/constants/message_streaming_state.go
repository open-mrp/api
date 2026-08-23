package constants

// MessageStreamingState is whether an agent reply's body is still being generated.
type MessageStreamingState string

const (
	// MessageStreamingStateStreaming means the body keeps growing as realtime updates arrive.
	MessageStreamingStateStreaming MessageStreamingState = "streaming"
	// MessageStreamingStateComplete means the body is final.
	MessageStreamingStateComplete MessageStreamingState = "complete"
)

func (s MessageStreamingState) IsValid() bool {
	switch s {
	case MessageStreamingStateStreaming, MessageStreamingStateComplete:
		return true
	default:
		return false
	}
}

func (s MessageStreamingState) EnumValues() []string {
	return []string{
		string(MessageStreamingStateStreaming),
		string(MessageStreamingStateComplete),
	}
}
