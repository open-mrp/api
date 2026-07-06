package domain

import (
	"time"

	"github.com/augno/api/shared/constants"
)

// PortalDNSRecord is a DNS record the customer must publish for their portal domain to route and verify.
type PortalDNSRecord struct {
	Type  constants.DNSRecordType `json:"type"`
	Name  string                  `json:"name"`
	Value string                  `json:"value"`
	// Reason explains why the record is needed: routing points traffic at the provider; ownership proves domain control when the domain is claimed elsewhere.
	Reason constants.DNSRecordReason `json:"reason"`
}

// PortalDomain represents a customer-supplied custom domain that serves the account's customer portal. Terminal provider rejections mark the domain failed; transient DNS misconfiguration stays pending.
type PortalDomain struct {
	ID         string
	AccountID  string
	Domain     string                       `audit:"domain"`
	Status     constants.PortalDomainStatus `audit:"status"`
	DNSRecords []PortalDNSRecord
	VerifiedAt *time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
