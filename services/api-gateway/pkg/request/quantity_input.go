package apirequest

// QuantityInput represents a value with an associated unit for create/update requests.
type QuantityInput struct {
	// The decimal value.
	Value string `json:"value" validate:"required" format:"decimal"`
	// The unit ID for the value.
	UnitID string `json:"unit_id" validate:"required,max=191"`
}
