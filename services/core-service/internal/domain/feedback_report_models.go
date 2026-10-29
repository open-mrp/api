package domain

import (
	"time"

	"github.com/open-mrp/api/shared/pagination"
)

type FeedbackReport struct {
	ID        string
	AccountID string  `audit:"account_id"`
	UserID    string  `audit:"user_id"`
	Message   string  `audit:"message"`
	Page      *string `audit:"page"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

type CreateFeedbackReportParams struct {
	AccountID string
	UserID    string
	Message   string
	Page      *string
}

type ListFeedbackReportsParams struct {
	AccountID string
	Cursor    *string
	Limit     int32
}

type ListFeedbackReportsResult struct {
	FeedbackReports []*FeedbackReport
	PageInfo        pagination.PageInfo
}
