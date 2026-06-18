package constants

// AccountTypeCode represents the type of an account. This is useful to specify when an account is either a standard account or a sandbox account.
type AccountTypeCode string

const (
	// AccountTypeCodeStandard indicates that the account is a standard account that is used for production and integration purposes.
	AccountTypeCodeStandard AccountTypeCode = "company" // ! NOTE: Should update to "standard" in DB and app code
	// AccountTypeCodeSandbox indicates that the account is a sandbox account that is used for testing and development purposes.
	AccountTypeCodeSandbox AccountTypeCode = "sandbox"
)

func (c AccountTypeCode) IsValid() bool {
	switch c {
	case AccountTypeCodeStandard, AccountTypeCodeSandbox:
		return true
	default:
		return false
	}
}

func (c AccountTypeCode) EnumValues() []string {
	return []string{string(AccountTypeCodeStandard), string(AccountTypeCodeSandbox)}
}
