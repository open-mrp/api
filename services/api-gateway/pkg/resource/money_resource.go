package apiresource

// A monetary amount.
type Money struct {
	// Decimal amount, as a string to preserve precision.
	Amount string `json:"amount" validate:"required" format:"decimal"`
	// Lowercase ISO 4217 currency code.
	Currency string `json:"currency" validate:"required"`
}
