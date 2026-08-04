package apirequest

// An amount together with the unit it is expressed in.
//
// The unit may be a currency, so money amounts such as a credit limit are written the same way as physical amounts like weights or counts.
type QuantityInput struct {
	// Decimal value, as a string to preserve precision.
	Value string `json:"value" validate:"required" format:"decimal"`
	// ID of the unit of measure for the value.
	UnitID string `json:"unit_id" validate:"required"`
}
