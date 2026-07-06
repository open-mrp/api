package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SamplePortalDomainID = "podn_018e88072d1320808dc9aab42"

// A DNS record the customer must publish for their portal domain.
type DNSRecord struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=dns_record"`
	// Record type.
	Type constants.DNSRecordType `json:"type" validate:"required"`
	// Record name (host) to publish.
	Name string `json:"name" validate:"required"`
	// Record value to publish.
	Value string `json:"value" validate:"required"`
	// Why the record is needed.
	//
	// Routing records point traffic at the portal's serving infrastructure; ownership records prove control of a domain that is claimed elsewhere.
	Reason constants.DNSRecordReason `json:"reason" validate:"required"`
}

// A custom domain that serves the account's customer portal (e.g. `shop.acme.com`).
//
// After creation the domain starts in `pending`; publish the returned DNS records, then poll the verify action until it flips to `verified`. Once verified, the customer portal is served on the domain with TLS provisioned automatically.
type PortalDomain struct {
	// Portal domain ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=portal_domain"`
	// The fully-qualified domain name (e.g. `shop.acme.com`).
	Domain string `json:"domain" validate:"required"`
	// Verification status.
	//
	// - pending domains await DNS configuration
	// - verified domains serve the portal
	// - failed domains were rejected and cannot be used
	Status constants.PortalDomainStatus `json:"status" validate:"required"`
	// The DNS records the customer must publish for the domain to route and verify.
	DNSRecords *List[DNSRecord] `json:"dns_records" validate:"required"`
	// When the domain's DNS configuration was confirmed.
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
	DNSRecords: NewList([]DNSRecord{
		{
			Object: constants.ObjectTypeDNSRecord,
			Type:   constants.DNSRecordTypeCNAME,
			Name:   "shop.acme.com",
			Value:  "cname.vercel-dns.com",
			Reason: constants.DNSRecordReasonRouting,
		},
	}, PageInfo{}),
	CreatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt: timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*PortalDomain) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SamplePortalDomain)
}
