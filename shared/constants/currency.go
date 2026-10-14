package constants

// Currency is an ISO 4217 currency code, lowercase.
type Currency string

const (
	// CurrencyUSD is the United States dollar, the only currency accepted at forge.1.
	CurrencyUSD Currency = "usd"
)

func (c Currency) IsValid() bool {
	switch c {
	case CurrencyUSD:
		return true
	default:
		return false
	}
}

func (c Currency) EnumValues() []string {
	return []string{string(CurrencyUSD)}
}

func (c *Currency) StringPtr() *string {
	if c == nil {
		return nil
	}
	s := string(*c)
	return &s
}
