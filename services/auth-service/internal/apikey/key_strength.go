package apikey

// KeyStrength describes the target entropy for generated keys:
// - low:    ~130 bits
// - medium: ~196 bits
// - high:   ~261 bits
type KeyStrength string

const (
	// KeyStrengthLow represents a key with 130 bits of entropy.
	KeyStrengthLow KeyStrength = "low"
	// KeyStrengthMedium represents a key with 196 bits of entropy.
	KeyStrengthMedium KeyStrength = "medium"
	// KeyStrengthHigh represents a key with 261 bits of entropy.
	KeyStrengthHigh KeyStrength = "high"
)
