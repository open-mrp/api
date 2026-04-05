package domain

import (
	"time"

	"github.com/augno/api/shared/pagination"
)

type EmailLog struct {
	ID           string
	HasSent      bool
	Recipients   []string
	Subject      *string
	Filename     *string
	SESMessageID *string
	SentByID     *string
	SentByName   *string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type ListEmailLogsParams struct {
	AccountID string
	Cursor    *string
	Limit     int32
	Query     *string
}

type ListEmailLogsResult struct {
	EmailLogs []*EmailLog
	PageInfo  pagination.PageInfo
}

type GetEmailLogParams struct {
	AccountID  string
	EmailLogID string
}
