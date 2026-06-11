package apirequest

// A rate value with its numerator and denominator units, used in create and update requests.
type RateInput struct {
	// Decimal value of the rate, expressed as the amount of the numerator unit per one denominator unit.
	Value string `json:"value" validate:"required" format:"decimal"`
	// ID of the unit for the rate's numerator (e.g. the currency of a price).
	NumeratorUnitID string `json:"numerator_unit_id" validate:"required"`
	// ID of the unit for the rate's denominator (the per-unit basis).
	DenominatorUnitID string `json:"denominator_unit_id" validate:"required"`
}
