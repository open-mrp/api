package apiresource

import (
	"time"

	"github.com/open-mrp/api/shared/constants"
	apiexample "github.com/open-mrp/apikit/example"
	"github.com/open-mrp/apikit/timeutil"
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
	// Who sent the email. Emails the platform sends on its own, such as system notifications, are sent by `system`.
	SentBy *ActorRef `json:"sent_by" validate:"required"`
	// The record the email carried, such as the invoice it attached. Emails logged before forge.1 have none.
	Related *EmailLogRelated `json:"related" validate:"required"`
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
	SentBy:     SampleActorRef,
	Related: &EmailLogRelated{
		Object: constants.ObjectTypeEmailLogRelated,
		Record: &Record{ID: SampleInvoiceID, Object: constants.ObjectTypeRecord, Type: constants.RecordTypeInvoice, Number: new("INV-1001"), Status: new("open"), Metadata: map[string]string{}},
	},
	CreatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt: timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

// The record an email carried, as a lightweight reference. Retrieve the record by its ID for full details.
type EmailLogRelated struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=email_log_related"`
	// The record the email carried: the invoice it attached, the order it acknowledged, or the customer a statement was for.
	Record *Record `json:"record" expandable:"true"`
}

func (*EmailLog) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleEmailLog)
}
