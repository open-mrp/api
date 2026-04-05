package constants

// AdjustmentType represents the type of an adjustment.
type AdjustmentType string

const (
	// AdjustmentTypeDiscount indicates that the adjustment is a discount.
	AdjustmentTypeDiscount AdjustmentType = "discount"
	// AdjustmentTypeShippingDiscrepancy indicates a shipping-related discrepancy.
	AdjustmentTypeShippingDiscrepancy AdjustmentType = "shipping_discrepancy"
	// AdjustmentTypeShortPayment indicates a short payment.
	AdjustmentTypeShortPayment AdjustmentType = "short_payment"
	// AdjustmentTypeWriteOff indicates a write-off.
	AdjustmentTypeWriteOff AdjustmentType = "write_off"
	// AdjustmentTypeFee indicates a fee adjustment.
	AdjustmentTypeFee AdjustmentType = "fee"
	// AdjustmentTypeRefund indicates a refund.
	AdjustmentTypeRefund AdjustmentType = "refund"
)

func (m AdjustmentType) IsValid() bool {
	switch m {
	case AdjustmentTypeDiscount, AdjustmentTypeShippingDiscrepancy, AdjustmentTypeShortPayment,
		AdjustmentTypeWriteOff, AdjustmentTypeFee, AdjustmentTypeRefund:
		return true
	default:
		return false
	}
}

func (m AdjustmentType) EnumValues() []string {
	return []string{
		string(AdjustmentTypeDiscount),
		string(AdjustmentTypeShippingDiscrepancy),
		string(AdjustmentTypeShortPayment),
		string(AdjustmentTypeWriteOff),
		string(AdjustmentTypeFee),
		string(AdjustmentTypeRefund),
	}
}
