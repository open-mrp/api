package constants

// PortalDomainStatus is the verification status of a customer portal custom domain.
type PortalDomainStatus string

const (
	// PortalDomainStatusPending indicates the domain is registered and awaiting correct DNS configuration.
	PortalDomainStatusPending PortalDomainStatus = "pending"
	// PortalDomainStatusVerified indicates the domain's DNS is confirmed and the portal is served on it.
	PortalDomainStatusVerified PortalDomainStatus = "verified"
	// PortalDomainStatusFailed indicates the domain was terminally rejected and cannot be used.
	PortalDomainStatusFailed PortalDomainStatus = "failed"
)

func (s PortalDomainStatus) IsValid() bool {
	switch s {
	case PortalDomainStatusPending, PortalDomainStatusVerified, PortalDomainStatusFailed:
		return true
	default:
		return false
	}
}

func (s PortalDomainStatus) EnumValues() []string {
	return []string{string(PortalDomainStatusPending), string(PortalDomainStatusVerified), string(PortalDomainStatusFailed)}
}

func (s *PortalDomainStatus) StringPtr() *string {
	if s == nil {
		return nil
	}
	v := string(*s)
	return &v
}

// DNSRecordType is the DNS record type a customer must publish for a portal domain.
type DNSRecordType string

const (
	// DNSRecordTypeCNAME points a subdomain at the portal's serving infrastructure.
	DNSRecordTypeCNAME DNSRecordType = "CNAME"
	// DNSRecordTypeA points an apex domain at the portal's serving infrastructure.
	DNSRecordTypeA DNSRecordType = "A"
	// DNSRecordTypeTXT carries an ownership-verification challenge.
	DNSRecordTypeTXT DNSRecordType = "TXT"
)

func (t DNSRecordType) IsValid() bool {
	switch t {
	case DNSRecordTypeCNAME, DNSRecordTypeA, DNSRecordTypeTXT:
		return true
	default:
		return false
	}
}

func (t DNSRecordType) EnumValues() []string {
	return []string{string(DNSRecordTypeCNAME), string(DNSRecordTypeA), string(DNSRecordTypeTXT)}
}

func (t *DNSRecordType) StringPtr() *string {
	if t == nil {
		return nil
	}
	v := string(*t)
	return &v
}

// DNSRecordReason explains why a portal domain DNS record must be published.
type DNSRecordReason string

const (
	// DNSRecordReasonRouting indicates the record points traffic at the portal's serving infrastructure.
	DNSRecordReasonRouting DNSRecordReason = "routing"
	// DNSRecordReasonOwnership indicates the record proves control of a domain that is claimed elsewhere.
	DNSRecordReasonOwnership DNSRecordReason = "ownership"
)

func (r DNSRecordReason) IsValid() bool {
	switch r {
	case DNSRecordReasonRouting, DNSRecordReasonOwnership:
		return true
	default:
		return false
	}
}

func (r DNSRecordReason) EnumValues() []string {
	return []string{string(DNSRecordReasonRouting), string(DNSRecordReasonOwnership)}
}

func (r *DNSRecordReason) StringPtr() *string {
	if r == nil {
		return nil
	}
	v := string(*r)
	return &v
}
