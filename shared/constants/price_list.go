package constants

// PriceListPriceType says how a price-list rule sets a price.
type PriceListPriceType string

const (
	// PriceListPriceTypeFixed sets exactly the rule's unit price.
	PriceListPriceTypeFixed PriceListPriceType = "fixed"
	// PriceListPriceTypePercentOfBase sets a percent of the item's base sales price.
	PriceListPriceTypePercentOfBase PriceListPriceType = "percent_of_base"
	// PriceListPriceTypeTiered sets a price by the total quantity ordered in the rule's scope.
	PriceListPriceTypeTiered PriceListPriceType = "tiered"
)

func (t PriceListPriceType) IsValid() bool {
	switch t {
	case PriceListPriceTypeFixed, PriceListPriceTypePercentOfBase, PriceListPriceTypeTiered:
		return true
	default:
		return false
	}
}

func (t PriceListPriceType) EnumValues() []string {
	return []string{string(PriceListPriceTypeFixed), string(PriceListPriceTypePercentOfBase), string(PriceListPriceTypeTiered)}
}

// PriceListSourceType names the legacy pricing record a price list mirrors while that record is still written.
type PriceListSourceType string

const (
	// PriceListSourceCustomer mirrors one customer's account prices (the customer's own list).
	PriceListSourceCustomer PriceListSourceType = "customer"
	// PriceListSourceAccountGroup mirrors one account group's volume discounts and price-group members.
	PriceListSourceAccountGroup PriceListSourceType = "account_group"
	// PriceListSourceAccount mirrors the account's volume discounts that name no customer group; it applies to every
	// customer.
	PriceListSourceAccount PriceListSourceType = "account"
)

func (t PriceListSourceType) IsValid() bool {
	switch t {
	case PriceListSourceCustomer, PriceListSourceAccountGroup, PriceListSourceAccount:
		return true
	default:
		return false
	}
}

func (t PriceListSourceType) EnumValues() []string {
	return []string{string(PriceListSourceCustomer), string(PriceListSourceAccountGroup), string(PriceListSourceAccount)}
}

func (t *PriceListPriceType) StringPtr() *string  { return stringPtrEnum(t) }
func (t *PriceListSourceType) StringPtr() *string { return stringPtrEnum(t) }
