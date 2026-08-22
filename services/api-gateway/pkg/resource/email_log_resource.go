package apiresource

import (
	"time"

	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/timeutil"
)

const SampleEmailLogID = "eml_h2j1q1nfibwb"

// A record of an email the platform sent on the account's behalf, such as an order acknowledgement or a user invitation.
//
// An email that never reached the delivery provider is recorded here too, rather than disappearing.
type EmailLog struct {
	// Email log ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=email_log"`
	// Whether the email was handed off to the delivery provider.
	//
	// - `sent`: the provider accepted the email for delivery. It does not confirm that the recipient's mail server accepted it.
	// - `pending`: the email was never handed off — the send attempt failed, or it was suppressed because the account is in sandbox mode.
	SendStatus constants.EmailSendStatus `json:"send_status" validate:"required"`
	// Recipient email addresses.
	Recipients []string `json:"recipients" validate:"required"`
	// Email subject line.
	Subject *string `json:"subject"`
	// Filename of the document attached to the email.
	Filename *string `json:"filename"`
	// The user or API key that sent the email.
	//
	// Emails the platform sends automatically, such as system notifications, are not attributed to an actor.
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
