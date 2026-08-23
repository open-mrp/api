package constants

// EmailDomainStatus is how far a customer-owned sending domain has gotten through DKIM verification.
type EmailDomainStatus string

const (
	// EmailDomainStatusPending means the DNS records have been issued but not yet verified.
	EmailDomainStatusPending EmailDomainStatus = "pending"
	// EmailDomainStatusVerified means the domain is verified and can send mail.
	EmailDomainStatusVerified EmailDomainStatus = "verified"
	// EmailDomainStatusFailed means verification did not complete.
	EmailDomainStatusFailed EmailDomainStatus = "failed"
)

func (s EmailDomainStatus) IsValid() bool {
	switch s {
	case EmailDomainStatusPending, EmailDomainStatusVerified, EmailDomainStatusFailed:
		return true
	default:
		return false
	}
}

func (s EmailDomainStatus) EnumValues() []string {
	return []string{
		string(EmailDomainStatusPending),
		string(EmailDomainStatusVerified),
		string(EmailDomainStatusFailed),
	}
}
