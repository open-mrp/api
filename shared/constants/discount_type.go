package constants

// DiscountType says how a discount is expressed.
type DiscountType string

const (
	// DiscountTypePercentOff takes a whole-number percent off.
	DiscountTypePercentOff DiscountType = "percent_off"
	// DiscountTypeAmountOff takes a fixed amount of money off.
	DiscountTypeAmountOff DiscountType = "amount_off"
)

func (d DiscountType) IsValid() bool {
	switch d {
	case DiscountTypePercentOff, DiscountTypeAmountOff:
		return true
	default:
		return false
	}
}

func (d DiscountType) EnumValues() []string {
	return []string{string(DiscountTypePercentOff), string(DiscountTypeAmountOff)}
}

func (d *DiscountType) StringPtr() *string {
	if d == nil {
		return nil
	}
	s := string(*d)
	return &s
}
