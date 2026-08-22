package domain

import (
	"time"

	"github.com/open-mrp/api/shared/pagination"
)

// EmailLogActor identifies the actor who sent an email. Today this is always a user, but the shape mirrors other actor references so future senders (API keys, agents) can be represented without another schema change.
type EmailLogActor struct {
	ID        string
	ActorType string
	Name      *string
	// Handle is the human-readable identifier for the actor — email for users, redacted value for API keys.
	Handle *string
}

type EmailLog struct {
	ID           string
	HasSent      bool
	Recipients   []string
	Subject      *string
	Filename     *string
	SESMessageID *string
	SentBy       *EmailLogActor
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type ListEmailLogsParams struct {
	AccountID string
	Cursor    *string
	Limit     int32
	Query     *string
	Includes  []string
}

type ListEmailLogsResult struct {
	EmailLogs []*EmailLog
	PageInfo  pagination.PageInfo
}

type GetEmailLogParams struct {
	AccountID  string
	EmailLogID string
	Includes   []string
}
