package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleEmailDomainID = "emdom_018e88072d1320808dc9aaa01"

// A domain registered with the email bridge for sending and receiving mail.
//
// After registration the domain starts in `pending`; publish the returned DKIM records, then poll the verify action until it flips to `verified`.
type EmailDomain struct {
	// Email domain ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=email_domain"`
	// The fully-qualified domain name (e.g. `support.acme.com`).
	Domain string `json:"domain" validate:"required"`
	// Verification status.
	//
	// - `pending`: registered and awaiting DKIM confirmation.
	// - `verified`: DKIM confirmed; the domain can send and receive mail.
	// - `failed`: verification could not be completed.
	Status string `json:"status" validate:"required"`
	// The DKIM CNAME tokens the customer must publish in DNS to verify the domain.
	DkimTokens []string `json:"dkim_tokens"`
	// When the domain's DKIM verification was confirmed.
	VerifiedAt *time.Time `json:"verified_at"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleEmailDomain = &EmailDomain{
	ID:         SampleEmailDomainID,
	Object:     constants.ObjectTypeEmailDomain,
	Domain:     "support.acme.com",
	Status:     "pending",
	DkimTokens: []string{"abc123._domainkey.support.acme.com", "def456._domainkey.support.acme.com"},
	CreatedAt:  timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:  timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*EmailDomain) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleEmailDomain)
}
