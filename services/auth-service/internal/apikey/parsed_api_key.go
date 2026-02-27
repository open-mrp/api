package apikey

import (
	"cmp"
	"strings"

	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/crypto"
	apierror "github.com/augno/api/shared/errors"
)

type ParsedAPIKey struct {
	AccountMode constants.AccountMode
	ID          string
	Secret      string // #nosec G117 - Struct field, not a hardcoded credential
	Checksum    string
}

func (p *ParsedAPIKey) String() string {
	return string(types.APIKeyPrefixSecretKey) + string(p.AccountMode) + "_" + p.ID + "_" + p.Secret + p.Checksum
}

// APIKeyGenConfig holds optional overrides for key generation strengths.
type APIKeyGenConfig struct {
	SecretKeyStrength KeyStrength
	IDKeyStrength     KeyStrength
}

const (
	apiKeyAugnoPrefixIdx = 0
	apiKeySKPrefixIdx    = 1
	apiKeyModeIdx        = 2
	apiKeyIDIdx          = 3
	apiKeySCSIdx         = 4
	apiKeyNumParts       = 5
	apiKeyChecksumLen    = 6
)

// GenParsedAPIKey generates a new parsed API key for the given account mode.
func GenParsedAPIKey(mode constants.AccountMode, config *APIKeyGenConfig) (*ParsedAPIKey, *apierror.APIError) {
	secretStrength := KeyStrengthHigh
	idStrength := KeyStrengthLow
	if config != nil {
		secretStrength = cmp.Or(config.SecretKeyStrength, KeyStrengthHigh)
		idStrength = cmp.Or(config.IDKeyStrength, KeyStrengthLow)
	}

	secret, err := crypto.RandAlphanumericString(secretStrength.Length())
	if err != nil {
		return nil, apierror.NewInternalError(err, "Failed to generate API Key.")
	}

	id, err := crypto.RandAlphanumericString(idStrength.Length())
	if err != nil {
		return nil, apierror.NewInternalError(err, "Failed to generate API Key.")
	}

	checksum := genAPIKeyChecksum(id, secret)

	return &ParsedAPIKey{
		AccountMode: mode,
		ID:          id,
		Secret:      secret,
		Checksum:    checksum,
	}, nil
}

// ParseAPIKey parses a raw API key string into its constituent parts.
func ParseAPIKey(key string) (*ParsedAPIKey, *apierror.APIError) {
	parts := strings.SplitN(key, "_", apiKeyNumParts)

	if len(parts) != apiKeyNumParts {
		return nil, invalidAPIKeyError(key)
	}

	if !strings.HasPrefix(parts[apiKeyAugnoPrefixIdx]+"_"+parts[apiKeySKPrefixIdx]+"_", string(types.APIKeyPrefixSecretKey)) {
		return nil, invalidAPIKeyError(key)
	}

	secretPlusChk := parts[apiKeySCSIdx]
	if len(secretPlusChk) < apiKeyChecksumLen {
		return nil, invalidAPIKeyError(key)
	}

	id := parts[apiKeyIDIdx]
	secret := secretPlusChk[:len(secretPlusChk)-apiKeyChecksumLen]
	chk := secretPlusChk[len(secretPlusChk)-apiKeyChecksumLen:]

	expected := genAPIKeyChecksum(id, secret)
	if expected != chk {
		return nil, invalidAPIKeyError(key)
	}

	appMode := parts[apiKeyModeIdx]
	if !constants.AccountMode(appMode).IsValid() {
		return nil, invalidAPIKeyError(key)
	}

	return &ParsedAPIKey{
		AccountMode: constants.AccountMode(appMode),
		ID:          id,
		Secret:      secret,
		Checksum:    chk,
	}, nil
}

// RedactedValue returns a display-safe representation of the full key string.
// Format: aug_sk_{mode}_****{last4}
func (p *ParsedAPIKey) RedactedValue() string {
	full := p.String()
	return string(types.APIKeyPrefixSecretKey) + string(p.AccountMode) + "_****" + full[len(full)-4:]
}

// GenSecretHMAC generates an HMAC-SHA256 for the parsed key's secret using the given pepper.
func (p *ParsedAPIKey) GenSecretHMAC(pepper []byte) []byte {
	return crypto.HMACSHA256(pepper, []byte(p.Secret))
}

// VerifySecretHMAC verifies the parsed key's secret against an expected HMAC.
func (p *ParsedAPIKey) VerifySecretHMAC(pepper, expectedHMAC []byte) bool {
	return crypto.VerifyHMACSHA256(pepper, []byte(p.Secret), expectedHMAC)
}
