package apikey

import (
	"fmt"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/crypto"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/sanitize"
)

// SanitizeAPIKey sanitizes a given API key string for display.
func SanitizeAPIKey(apiKey string) string {
	return sanitize.SanitizeString(apiKey, len(types.APIKeyPrefixSecretKey)+len(constants.AccountModeProduction)+3, 4)
}

func invalidAPIKeyError(key string) *apierror.APIError {
	message := fmt.Sprintf("%s: %s. Valid API keys start with '%s'.", ErrAPIKeyInvalid, SanitizeAPIKey(key), string(types.APIKeyPrefixSecretKey))
	return apierror.NewInvalidFormatError(message, "api_key")
}

func genAPIKeyChecksum(id, secret string) string {
	return crypto.CRC32Base62(id+"_"+secret, apiKeyChecksumLen)
}

// KeyStrength describes the target entropy for generated keys:
// - low:    ~130 bits
// - medium: ~196 bits
// - high:   ~261 bits
type KeyStrength string

const (
	KeyStrengthLow    KeyStrength = "low"
	KeyStrengthMedium KeyStrength = "medium"
	KeyStrengthHigh   KeyStrength = "high"
)

// Length returns the length of the key for the given strength.
func (k *KeyStrength) Length() int {
	if k == nil {
		return 44
	}

	switch *k {
	case KeyStrengthLow:
		return 22
	case KeyStrengthMedium:
		return 33
	case KeyStrengthHigh:
		return 44
	default:
		return 44
	}
}
