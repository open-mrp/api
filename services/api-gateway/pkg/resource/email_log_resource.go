package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleEmailLogID = "eml_017b80707ada92dddff8a2c3a0"

// A record of an email sent on the account's behalf, such as an invoice or a user invitation.
type EmailLog struct {
	// Email log ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=email_log"`
	// Email send status.
	//
	// - `pending`: the email is queued and has not been sent yet.
	// - `sent`: the email has been handed off for delivery.
	SendStatus constants.EmailSendStatus `json:"send_status" validate:"required"`
	// Recipient email addresses.
	Recipients []string `json:"recipients" validate:"required"`
	// Email subject line.
	Subject *string `json:"subject"`
	// Filename of the attached document.
	Filename *string `json:"filename"`
	// The user who sent the email.
	//
	// Absent for emails the platform sends automatically (for example system notifications), which are not attributed to a user.
	SentBy *Actor `json:"sent_by" expandable:"true"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var sampleEmailLogSubject = "Order Confirmation #1001"
var sampleEmailLogFilename = "invoice_1001.pdf"

var SampleEmailLog = &EmailLog{
	ID:         SampleEmailLogID,
	Object:     constants.ObjectTypeEmailLog,
	SendStatus: constants.EmailSendStatusSent,
	Recipients: []string{"customer@example.com"},
	Subject:    &sampleEmailLogSubject,
	Filename:   &sampleEmailLogFilename,
	SentBy:     SampleActor,
	CreatedAt:  timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:  timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*EmailLog) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleEmailLog)
}
