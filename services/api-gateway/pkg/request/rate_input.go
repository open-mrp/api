package apirequest

// A value expressed as a ratio of two units, supplied on create and update requests.
//
// A unit price, for example, has a currency as its numerator unit and the unit the product is bought or sold by as its denominator.
type RateInput struct {
	// Decimal value of the rate, expressed as the amount of the numerator unit per one denominator unit.
	Value string `json:"value" validate:"required" format:"decimal"`
	// ID of the unit for the rate's numerator (e.g. the currency of a price).
	NumeratorUnitID string `json:"numerator_unit_id" validate:"required"`
	// ID of the unit for the rate's denominator (the per-unit basis).
	DenominatorUnitID string `json:"denominator_unit_id" validate:"required"`
}
