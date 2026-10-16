package apiresource

import (
	"time"

	"github.com/open-mrp/api/shared/constants"
	apiexample "github.com/open-mrp/apikit/example"
	"github.com/open-mrp/apikit/timeutil"
)

const SamplePortalDomainID = "podn_ml44z5ggf169"

// A DNS record to publish for a portal domain.
type PortalDNSRecord struct {
	// The record type.
	//
	// - `CNAME`: points a subdomain at the portal.
	// - `A`: points an apex domain at the portal.
	// - `TXT`: proves you control a domain that is already in use elsewhere.
	Type constants.DNSRecordType `json:"type" validate:"required"`
	// The record name (host) to publish.
	Name string `json:"name" validate:"required"`
	// The record value to publish.
	Value string `json:"value" validate:"required"`
}

// A custom domain for your customer portal.
//
// Publish `dns_records` at your DNS provider. The domain moves from `pending` to `securing` while a certificate is issued, then to `verified`. It's checked automatically; Verify checks now.
type PortalDomain struct {
	// Portal domain ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=portal_domain"`
	// The fully qualified domain name, such as `shop.acme.com`. Unique across OpenMRP, case-insensitive.
	Domain string `json:"domain" validate:"required"`
	// How far the domain is towards serving your portal.
	//
	// - `pending`: waiting for the DNS records to be published. Pending domains are checked automatically.
	// - `securing`: the records are in place and the TLS certificate is being issued. The portal isn't reachable over HTTPS yet.
	// - `verified`: the portal is served on the domain.
	// - `failed`: the domain was rejected and can't be used.
	Status constants.PortalDomainStatus `json:"status" validate:"required"`
	// The DNS records to publish at your DNS provider.
	//
	// Always includes the routing record (`CNAME` for a subdomain, `A` for an apex domain); a `TXT` record appears while ownership of a domain used elsewhere must be proven.
	DNSRecords []PortalDNSRecord `json:"dns_records" validate:"required"`
	// When the domain was verified.
	VerifiedAt *time.Time `json:"verified_at"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SamplePortalDomain = &PortalDomain{
	ID:     SamplePortalDomainID,
	Object: constants.ObjectTypePortalDomain,
	Domain: "shop.acme.com",
	Status: constants.PortalDomainStatusPending,
	DNSRecords: []PortalDNSRecord{
		{Type: constants.DNSRecordTypeCNAME, Name: "shop.acme.com", Value: "cname.vercel-dns.com"},
	},
	CreatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt: timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*PortalDomain) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SamplePortalDomain)
}
