package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleEmailLogID = "eml_01jm4r6700f8nwq3v5hx2d9ktp"

// EmailLog represents an email log entry.
type EmailLog struct {
	// The unique identifier for the email log.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=email_log"`
	// Whether the email was sent.
	HasSent bool `json:"has_sent"`
	// The recipient email addresses.
	Recipients []string `json:"recipients" validate:"required"`
	// The email subject line.
	Subject *string `json:"subject"`
	// The filename of any attachment.
	Filename *string `json:"filename"`
	// The SES message ID returned by AWS.
	SESMessageID *string `json:"ses_message_id"`
	// The actor who sent the email. Null when the email was sent by the
	// system, or when the caller did not request `include=sent_by`.
	SentBy *Actor `json:"sent_by" expandable:"true"`
	// When this email log was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// When this email log was last updated.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var sampleEmailLogSubject = "Order Confirmation #1001"
var sampleEmailLogFilename = "invoice_1001.pdf"

var SampleEmailLog = &EmailLog{
	ID:         SampleEmailLogID,
	Object:     constants.ObjectTypeEmailLog,
	HasSent:    true,
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
