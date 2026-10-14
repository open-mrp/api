package apirequest

// A value measured per another unit, such as `100 kg / hr`.
type RateInput struct {
	// Decimal value, as a string to keep precision.
	Value string `json:"value" validate:"required" format:"decimal"`
	// ID of the unit the value is measured in, such as kilograms in `100 kg / hr`.
	UnitID string `json:"unit_id" validate:"required"`
	// ID of the unit the rate is measured per, such as hours in `100 kg / hr`.
	PerUnitID string `json:"per_unit_id" validate:"required"`
}
