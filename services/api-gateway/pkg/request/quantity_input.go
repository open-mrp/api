package apirequest

// A value with an associated unit, used in create and update requests.
type QuantityInput struct {
	// Decimal value, as a string to preserve precision.
	Value string `json:"value" validate:"required" format:"decimal"`
	// ID of the unit of measure for the value.
	UnitID string `json:"unit_id" validate:"required"`
}
