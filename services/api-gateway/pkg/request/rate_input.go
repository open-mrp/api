package apirequest

// RateInput represents a rate (value with numerator and denominator units) for
// create requests. Mirrors the gRPC CreateRateInput message — when a caller
// passes a cost-typed rate (unit_price, unit_cost, labor_rate, overhead_rate),
// the core service enforces that the numerator unit is currency and the
// denominator unit is not.
type RateInput struct {
	// Decimal value of the rate.
	Value string `json:"value" validate:"required" format:"decimal"`
	// Numerator unit ID (e.g. the "$" in "$5/ea").
	NumeratorUnitID string `json:"numerator_unit_id" validate:"required,max=191"`
	// Denominator unit ID (e.g. the "ea" in "$5/ea").
	DenominatorUnitID string `json:"denominator_unit_id" validate:"required,max=191"`
}
