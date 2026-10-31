package apiresource

import (
	"time"

	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/timeutil"
)

const SampleFeedbackReportID = "fbro_cyyu7rmo73nq"

// Feedback or a bug report a user submitted from the dashboard.
type FeedbackReport struct {
	// Feedback report ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=feedback_report"`
	// What the user reported, in their own words.
	Message string `json:"message" validate:"required"`
	// Dashboard page the user was on when they submitted the report.
	Page *string `json:"page"`
	// How urgently the report needs attention.
	Severity constants.FeedbackReportSeverity `json:"severity" validate:"required"`
	// User who submitted the report.
	User *User `json:"user" expandable:"true"`
	// When the report was submitted.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// When the report was last updated.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var sampleFeedbackReportPage = "/dashboard/items"

var SampleFeedbackReport = &FeedbackReport{
	ID:        SampleFeedbackReportID,
	Object:    constants.ObjectTypeFeedbackReport,
	Message:   "The item list loses its filters when I open an item and come back.",
	Page:      &sampleFeedbackReportPage,
	Severity:  constants.FeedbackReportSeverityMedium,
	User:      nil,
	CreatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt: timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*FeedbackReport) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleFeedbackReport)
}
