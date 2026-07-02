package constants

// ScheduledMessageStatus is the lifecycle state of a scheduled message. It is an enum so new states can be added without a breaking change to the API.
type ScheduledMessageStatus string

const (
	// ScheduledMessageStatusPending is queued for delivery at its scheduled time.
	ScheduledMessageStatusPending ScheduledMessageStatus = "pending"
	// ScheduledMessageStatusSent was delivered (sent_message_id points at the resulting message).
	ScheduledMessageStatusSent ScheduledMessageStatus = "sent"
	// ScheduledMessageStatusCanceled was canceled before delivery (by the user, or because the conversation/sender was no longer valid at send time).
	ScheduledMessageStatusCanceled ScheduledMessageStatus = "canceled"
	// ScheduledMessageStatusFailed exhausted delivery attempts.
	ScheduledMessageStatusFailed ScheduledMessageStatus = "failed"
)

func (s ScheduledMessageStatus) IsValid() bool {
	switch s {
	case ScheduledMessageStatusPending, ScheduledMessageStatusSent, ScheduledMessageStatusCanceled, ScheduledMessageStatusFailed:
		return true
	default:
		return false
	}
}

func (s ScheduledMessageStatus) EnumValues() []string {
	return []string{
		string(ScheduledMessageStatusPending),
		string(ScheduledMessageStatusSent),
		string(ScheduledMessageStatusCanceled),
		string(ScheduledMessageStatusFailed),
	}
}

func (s *ScheduledMessageStatus) StringPtr() *string {
	if s == nil {
		return nil
	}
	v := string(*s)
	return &v
}
