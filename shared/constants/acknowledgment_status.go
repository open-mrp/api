package constants

// AcknowledgmentStatus represents whether an order acknowledgment has been sent to the customer. Modeled as an enum (rather than a bool) so additional states can be added later without a breaking change.
type AcknowledgmentStatus string

const (
	// AcknowledgmentStatusNotSent indicates no acknowledgment has been sent.
	AcknowledgmentStatusNotSent AcknowledgmentStatus = "not_sent"
	// AcknowledgmentStatusSent indicates the acknowledgment has been sent.
	AcknowledgmentStatusSent AcknowledgmentStatus = "sent"
)

func (m AcknowledgmentStatus) IsValid() bool {
	switch m {
	case AcknowledgmentStatusNotSent, AcknowledgmentStatusSent:
		return true
	default:
		return false
	}
}

func (m AcknowledgmentStatus) EnumValues() []string {
	return []string{
		string(AcknowledgmentStatusNotSent),
		string(AcknowledgmentStatusSent),
	}
}
