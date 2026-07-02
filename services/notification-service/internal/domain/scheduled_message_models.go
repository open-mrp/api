package domain

import "time"

// CreateScheduledMessageInput is the validated input for scheduling a message for future delivery.
// A scheduled message is a message row at status=scheduled; a lease-guarded worker promotes it to a
// sent timeline message in place when ScheduledFor arrives.
type CreateScheduledMessageInput struct {
	ConversationID string
	Body           string
	ScheduledFor   time.Time
}
