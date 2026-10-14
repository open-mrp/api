package apiresource

import (
	"time"

	"github.com/open-mrp/api/shared/constants"
	apiexample "github.com/open-mrp/apikit/example"
	"github.com/open-mrp/apikit/timeutil"
)

const SampleEmailSenderID = "emsx_4bk9pmt2wvqd"

// The address your order, invoice, and statement emails are sent from.
//
// Configure one on a verified email domain and your customers see mail from your own address instead of the platform's. Emails about someone's OpenMRP account — password resets, verification, plan changes — always send from the platform address regardless of this setting.
//
// Mail only sends from this address while the underlying domain stays verified; if verification lapses it falls back to the platform address rather than failing to send.
type EmailSender struct {
	// Email sender ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=email_sender"`
	// The email domain this address belongs to.
	//
	// Mail only sends from this address while the domain is `verified`.
	EmailDomain *EmailDomain `json:"email_domain" expandable:"true"`
	// The mailbox name before the `@`.
	LocalPart string `json:"local_part" validate:"required"`
	// The full sending address.
	Address string `json:"address" validate:"required"`
	// The name shown in a mail client's sender column. When unset, mail shows the bare address.
	FromName *string `json:"from_name"`
	// Where customer replies are delivered. When unset, replies go to the sending address.
	ReplyTo *string `json:"reply_to"`

	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleEmailSender = &EmailSender{
	ID:          SampleEmailSenderID,
	Object:      constants.ObjectTypeEmailSender,
	EmailDomain: SampleEmailDomain,
	LocalPart:   "orders",
	Address:     "orders@support.acme.com",
	CreatedAt:   timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:   timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*EmailSender) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleEmailSender)
}
