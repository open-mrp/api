package constants

// EmailDNSRecordType is the type of a DNS record a customer publishes for an email domain.
type EmailDNSRecordType string

const (
	// EmailDNSRecordTypeCNAME delegates a DKIM signing key.
	EmailDNSRecordTypeCNAME EmailDNSRecordType = "CNAME"
	// EmailDNSRecordTypeMX receives bounces for the MAIL FROM subdomain.
	EmailDNSRecordTypeMX EmailDNSRecordType = "MX"
	// EmailDNSRecordTypeTXT authorizes sending for the MAIL FROM subdomain (SPF).
	EmailDNSRecordTypeTXT EmailDNSRecordType = "TXT"
)

func (t EmailDNSRecordType) IsValid() bool {
	switch t {
	case EmailDNSRecordTypeCNAME, EmailDNSRecordTypeMX, EmailDNSRecordTypeTXT:
		return true
	default:
		return false
	}
}

func (t EmailDNSRecordType) EnumValues() []string {
	return []string{string(EmailDNSRecordTypeCNAME), string(EmailDNSRecordTypeMX), string(EmailDNSRecordTypeTXT)}
}

func (t *EmailDNSRecordType) StringPtr() *string {
	if t == nil {
		return nil
	}
	v := string(*t)
	return &v
}
