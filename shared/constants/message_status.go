package constants

// MessageStatus is the lifecycle state of a message. A message is the single resource for sent,
// scheduled, and draft content. Only Sent messages occupy the conversation timeline (and carry a
// sequence); Draft and Scheduled are unsent rows promoted to Sent in place. The remaining states are
// terminal. It is an enum so new states can be added without a breaking change to the API.
type MessageStatus string

const (
	// MessageStatusDraft is an editable customer-reply draft awaiting approval (not in the timeline).
	MessageStatusDraft MessageStatus = "draft"
	// MessageStatusScheduled is queued for delivery at a future time (not yet in the timeline).
	MessageStatusScheduled MessageStatus = "scheduled"
	// MessageStatusSent is a delivered timeline message (has a sequence).
	MessageStatusSent MessageStatus = "sent"
	// MessageStatusCanceled is a scheduled message canceled before delivery (terminal).
	MessageStatusCanceled MessageStatus = "canceled"
	// MessageStatusRejected is a draft discarded without sending (terminal).
	MessageStatusRejected MessageStatus = "rejected"
	// MessageStatusFailed is a scheduled message that exhausted delivery attempts (terminal).
	MessageStatusFailed MessageStatus = "failed"
	// MessageStatusSuperseded is a draft replaced by a newer one for the same source thread (terminal).
	MessageStatusSuperseded MessageStatus = "superseded"
)

func (s MessageStatus) IsValid() bool {
	switch s {
	case MessageStatusDraft, MessageStatusScheduled, MessageStatusSent,
		MessageStatusCanceled, MessageStatusRejected, MessageStatusFailed, MessageStatusSuperseded:
		return true
	default:
		return false
	}
}

func (s MessageStatus) EnumValues() []string {
	return []string{
		string(MessageStatusDraft),
		string(MessageStatusScheduled),
		string(MessageStatusSent),
		string(MessageStatusCanceled),
		string(MessageStatusRejected),
		string(MessageStatusFailed),
		string(MessageStatusSuperseded),
	}
}

func (s *MessageStatus) StringPtr() *string {
	if s == nil {
		return nil
	}
	v := string(*s)
	return &v
}
