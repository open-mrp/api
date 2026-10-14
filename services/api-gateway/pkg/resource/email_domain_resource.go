package apiresource

import (
	"time"

	"github.com/open-mrp/api/shared/constants"
	apiexample "github.com/open-mrp/apikit/example"
	"github.com/open-mrp/apikit/timeutil"
)

const SampleEmailDomainID = "emdom_2rk3omr8vshb"

// A domain you send email from.
//
// Publish `dns_records` at your DNS provider. Verification runs automatically; Verify checks now.
type EmailDomain struct {
	// Email domain ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=email_domain"`
	// The fully qualified domain name, such as `support.acme.com`. Unique across OpenMRP, case-insensitive.
	Domain string `json:"domain" validate:"required"`
	// Verification status.
	//
	// - `pending`: waiting for the DNS records to be published. Pending domains are checked automatically.
	// - `verified`: the records are in place; the domain can send mail.
	// - `failed`: verification could not be completed.
	//
	// Inboxes can only be created on a `verified` domain.
	Status constants.EmailDomainStatus `json:"status" validate:"required"`
	// The DNS records to publish at your DNS provider: one CNAME per DKIM signing key, then the MX and TXT (SPF) records of the bounce subdomain.
	//
	// Until the bounce records are published, mail clients may show your mail as sent "via amazonses.com".
	DNSRecords []EmailDNSRecord `json:"dns_records" validate:"required"`
	// When the domain was verified.
	VerifiedAt *time.Time `json:"verified_at"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

// A DNS record to publish for an email domain.
type EmailDNSRecord struct {
	// The record type: `CNAME`, `MX` or `TXT`.
	Type constants.EmailDNSRecordType `json:"type" validate:"required"`
	// The record name (host) to publish.
	Name string `json:"name" validate:"required"`
	// The record value to publish.
	Value string `json:"value" validate:"required"`
}

var SampleEmailDomain = &EmailDomain{
	ID:     SampleEmailDomainID,
	Object: constants.ObjectTypeEmailDomain,
	Domain: "support.acme.com",
	Status: constants.EmailDomainStatusPending,
	DNSRecords: []EmailDNSRecord{
		{Type: constants.EmailDNSRecordTypeCNAME, Name: "abc123._domainkey.support.acme.com", Value: "abc123.dkim.amazonses.com"},
		{Type: constants.EmailDNSRecordTypeMX, Name: "mail.support.acme.com", Value: "10 feedback-smtp.us-east-1.amazonses.com"},
		{Type: constants.EmailDNSRecordTypeTXT, Name: "mail.support.acme.com", Value: "v=spf1 include:amazonses.com ~all"},
	},
	CreatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt: timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*EmailDomain) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleEmailDomain)
}
