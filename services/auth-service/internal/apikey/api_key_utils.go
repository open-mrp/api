package apikey

import (
	"cmp"
	"context"
	"fmt"
	"strings"

	"github.com/augno/api/services/auth-service/internal/domain"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/crypto"
	apierror "github.com/augno/api/shared/errors"
	sanitize "github.com/augno/api/shared/sanitize"
	"github.com/augno/api/shared/tracing"

	"go.opentelemetry.io/otel/trace"
)

var apiKeyUtilsTracer = tracing.GetTracer("auth-service.api_key_utils")

// APIKeyConfig holds the configuration for API key generation and validation.
type APIKeyConfig struct {
	// SecretKeyStrength (optional; default: KeyStrengthHigh) is the target entropy for generated secret keys.
	SecretKeyStrength KeyStrength

	// IDKeyStrength (optional; default: KeyStrengthLow) is the target entropy for generated ID keys.
	IDKeyStrength KeyStrength

	// Pepper (required) is the additional secret mixed into API key hashes.
	Pepper []byte
}

// WithDefaults returns a new APIKeyConfig with zero-value fields replaced by production defaults.
func (c *APIKeyConfig) WithDefaults() *APIKeyConfig {
	if c == nil {
		c = &APIKeyConfig{}
	}

	return &APIKeyConfig{
		SecretKeyStrength: cmp.Or(c.SecretKeyStrength, KeyStrengthHigh),
		IDKeyStrength:     cmp.Or(c.IDKeyStrength, KeyStrengthLow),
		Pepper:            c.Pepper,
	}
}

// validate checks that all required APIKeyConfig fields are set.
func (c *APIKeyConfig) validate() error {
	if c.Pepper == nil {
		return fmt.Errorf("apikey: pepper is required")
	}
	return nil
}

type apiKeyUtilsImpl struct {
	config APIKeyConfig
}

// NewAPIKeyUtils creates a new API key utility with the given configuration.
func NewAPIKeyUtils(config *APIKeyConfig) domain.APIKeyUtils {
	config = config.WithDefaults()
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &apiKeyUtilsImpl{config: *config}
}

// Gen generates a new API key for the given account mode.
func (aku *apiKeyUtilsImpl) Gen(ctx context.Context, appMode constants.AccountMode) (*domain.ParsedAPIKey, *apierror.APIError) {
	_, span := apiKeyUtilsTracer.Start(ctx, "utils.api_key.generate")
	defer span.End()

	// Generate a random secret key of the given strength
	secret, err := genRandString(aku.lengthForKeyStrength(aku.config.SecretKeyStrength))
	if err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to generate API Key."))
	}

	// Generate a random id key of the given strength
	id, err := genRandString(aku.lengthForKeyStrength(aku.config.IDKeyStrength))
	if err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to generate API Key."))
	}

	// Generate a checksum for the key using the id and secret
	checksum := genKeyChecksum(id, secret)

	parsedKey := domain.ParsedAPIKey{
		AccountMode: appMode,
		ID:          id,
		Secret:      secret,
		Checksum:    checksum,
	}

	return &parsedKey, nil
}

// Parse parses a given API key string into its components.
func (aku *apiKeyUtilsImpl) Parse(ctx context.Context, key string) (*domain.ParsedAPIKey, *apierror.APIError) {
	const (
		augnoPrefixIdx = 0 // the indext of the `augno` prefix
		sKPrefixIdx    = 1 // the indext of the `sk` prefix
		modeIdx        = 2 // index of the account mode in the key
		idIdx          = 3 // index of the id in the key
		scsIdx         = 4 // index of the secret and checksum in the key
		numKeyParts    = 5 // number of parts in the key
		checksumLen    = 6 // checksum length should be atleast 7 characters
	)

	_, span := apiKeyUtilsTracer.Start(ctx, "utils.api_key.parse")
	defer span.End()

	parts := strings.SplitN(key, "_", numKeyParts)

	// We expect there to be parts to the key
	if len(parts) != numKeyParts {
		return nil, aku.invalidAPIKeyError(span, key)
	}

	// The first two parts of the key should be the secret key prefix
	if !strings.HasPrefix(parts[augnoPrefixIdx]+"_"+parts[sKPrefixIdx]+"_", string(types.APIKeyPrefixSecretKey)) {
		return nil, aku.invalidAPIKeyError(span, key)
	}

	// The last part of the key should be at least 7 characters
	secretPlusChk := parts[scsIdx]
	if len(secretPlusChk) < checksumLen {
		return nil, aku.invalidAPIKeyError(span, key)
	}

	// Break down the key into the key elements
	id := parts[idIdx]
	secret := secretPlusChk[:len(secretPlusChk)-checksumLen]
	chk := secretPlusChk[len(secretPlusChk)-checksumLen:]

	// If the checksum doesn't match, we know we didn't mint this key and we can fail fast
	expected := genKeyChecksum(id, secret)
	if expected != chk {
		return nil, aku.invalidAPIKeyError(span, key)
	}

	// The middle part of the key should be the account mode
	appMode := parts[modeIdx]
	if !constants.AccountMode(appMode).IsValid() {
		return nil, aku.invalidAPIKeyError(span, key)
	}

	parsedKey := domain.ParsedAPIKey{
		AccountMode: constants.AccountMode(appMode),
		ID:          id,
		Secret:      secret,
		Checksum:    chk,
	}

	return &parsedKey, nil
}

// GenSecretHMAC generates a HMAC for a given secret.
func (aku *apiKeyUtilsImpl) GenSecretHMAC(ctx context.Context, secret string) ([]byte, *apierror.APIError) {
	_, span := apiKeyUtilsTracer.Start(ctx, "utils.api_key.generate_secret_hmac")
	defer span.End()

	return crypto.HMACSHA256(aku.config.Pepper, []byte(secret)), nil
}

// VerifySecretHMAC verifies a given secret against a expected HMAC.
func (aku *apiKeyUtilsImpl) VerifySecretHMAC(ctx context.Context, secret string, expectedHMAC []byte) (bool, *apierror.APIError) {
	_, span := apiKeyUtilsTracer.Start(ctx, "utils.api_key.verify_secret_hmac")
	defer span.End()

	return crypto.VerifyHMACSHA256(aku.config.Pepper, []byte(secret), expectedHMAC), nil
}

// SanitizeForDisplay sanitizes a given API key for display.
//
// Ex: aug_sk_prod_aG****UCZu
func (aku *apiKeyUtilsImpl) SanitizeForDisplay(apiKey string) string {
	return sanitize.SanitizeString(apiKey, len(types.APIKeyPrefixSecretKey)+len(constants.AccountModeProduction)+3, 4)
}

func (aku *apiKeyUtilsImpl) lengthForKeyStrength(strength KeyStrength) int {
	switch strength {
	case KeyStrengthLow:
		return 22
	case KeyStrengthMedium:
		return 33
	case KeyStrengthHigh:
		return 44
	}
	return 44 // as default keystrength is high
}

func (aku *apiKeyUtilsImpl) invalidAPIKeyError(span trace.Span, key string) *apierror.APIError {
	message := fmt.Sprintf("%s: %s. Valid API keys start with '%s'.", ErrAPIKeyInvalid, aku.SanitizeForDisplay(key), string(types.APIKeyPrefixSecretKey))
	return tracing.Trace(span, apierror.NewInvalidFormatError(message, "api_key"))
}

// RedactAPIKeyValue redacts a given API key value for display.
//
//  1. Redacts the API key value for display.
//  2. Returns the redacted API key value.
func RedactAPIKeyValue(apiKeyModel *domain.APIKey, appMode constants.AccountMode) string {
	return string(types.APIKeyPrefixSecretKey) + string(appMode) + "_" + "****" + apiKeyModel.LastFour
}
