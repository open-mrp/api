package domain

import (
	"time"
)

type EmailLog struct {
	ID           string
	HasSent      bool
	AccountID    string
	SentByID     *string
	Subject      *string
	Filename     *string
	SesMessageID *string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
