package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SamplePortalDomainID = "podn_ml44z5ggf169"

// A DNS record that must be published at your DNS provider before a portal domain can be verified and serve traffic.
type DNSRecord struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=dns_record"`
	// The kind of DNS record to publish.
	//
	// - `CNAME`: points a subdomain at the portal's serving infrastructure.
	// - `A`: points an apex domain at the portal's serving infrastructure.
	// - `TXT`: carries an ownership-verification challenge.
	Type constants.DNSRecordType `json:"type" validate:"required"`
	// Record name (host) to publish.
	Name string `json:"name" validate:"required"`
	// Record value to publish.
	Value string `json:"value" validate:"required"`
	// Why the record must be published.
	//
	// - `routing`: the record points traffic at the portal's serving infrastructure.
	// - `ownership`: the record proves control of a domain that is already claimed elsewhere.
	Reason constants.DNSRecordReason `json:"reason" validate:"required"`
}

// A custom domain that serves the account's customer portal (e.g. `shop.acme.com`).
//
// After creation the domain starts in `pending`; publish the returned DNS records, then poll the verify action. Once DNS is correct the domain moves to `securing` while its TLS certificate is issued — it is not yet reachable over HTTPS during this window — and finally to `verified` once the certificate is live and the portal is served on the domain.
type PortalDomain struct {
	// Portal domain ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=portal_domain"`
	// The fully-qualified domain name (e.g. `shop.acme.com`).
	Domain string `json:"domain" validate:"required"`
	// How far the domain has progressed towards serving the portal.
	//
	// - `pending`: the domain is waiting on DNS. Publish the listed records, then run the verify action.
	// - `securing`: DNS is correct and the TLS certificate is being issued. The portal is not yet reachable over HTTPS.
	// - `verified`: the certificate is live and the portal is served on the domain.
	// - `failed`: the domain was rejected and cannot be used.
	Status constants.PortalDomainStatus `json:"status" validate:"required"`
	// The DNS records that must be published for the domain to route to the portal and verify.
	//
	// The list is refreshed from the serving provider every time the domain is created or verified. It always contains the routing record; ownership records appear only while a verification challenge is outstanding.
	DNSRecords *List[DNSRecord] `json:"dns_records" validate:"required"`
	// When the domain became fully verified — its TLS certificate live and the portal serving on it.
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
