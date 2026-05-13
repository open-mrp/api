package apirequest

type RateInput struct {
	// Decimal value of the rate.
	Value string `json:"value" validate:"required" format:"decimal"`
	// Numerator unit ID.
	NumeratorUnitID string `json:"numerator_unit_id" validate:"required"`
	// Denominator unit ID.
	DenominatorUnitID string `json:"denominator_unit_id" validate:"required"`
}
