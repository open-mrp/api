package apirequest

// A measured amount: a value and the unit it is measured in.
type QuantityInput struct {
	// Decimal value, as a string to keep precision.
	Value string `json:"value" validate:"required" format:"decimal"`
	// ID of the unit the value is measured in.
	UnitID string `json:"unit_id" validate:"required"`
}
