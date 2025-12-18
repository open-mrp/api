package apikey

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"strings"

	"github.com/augno/api/services/auth-service/internal/domain"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/contracts"
	sanitize "github.com/augno/api/shared/sanitize"
	"github.com/augno/api/shared/tracing"

	"go.opentelemetry.io/otel/trace"
)

var apiKeyUtilsTracer = tracing.GetTracer("auth-service.api_key_utils")

type APIKeyConfig struct {
	SecretKeyStrength KeyStrength
	IDKeyStrength     KeyStrength
	Pepper            []byte
}

func DefaultAPIKeyConfig(pepper []byte) APIKeyConfig {
	return APIKeyConfig{
		SecretKeyStrength: KeyStrengthHigh,
		IDKeyStrength:     KeyStrengthLow,
		Pepper:            pepper,
	}
}

type apiKeyUtilsImpl struct {
	config APIKeyConfig
}

func NewAPIKeyUtils(config APIKeyConfig) domain.APIKeyUtils {
	if config.Pepper == nil {
		panic("Pepper is not set in the config.")
	}

	return &apiKeyUtilsImpl{config: config}
}

// Gen generates a new API key for the given account mode.
func (aku *apiKeyUtilsImpl) Gen(ctx context.Context, appMode constants.AccountMode) (*domain.ParsedAPIKey, *contracts.APIError) {
	_, span := apiKeyUtilsTracer.Start(ctx, "utils.api_key.generate")
	defer span.End()

	// Generate a random secret key of the given strength
	secret, err := genRandString(aku.lengthForKeyStrength(aku.config.SecretKeyStrength))
	if err != nil {
		return nil, tracing.Trace(span, contracts.NewInternalError(err, "Failed to generate API Key."))
	}

	// Generate a random id key of the given strength
	id, err := genRandString(aku.lengthForKeyStrength(aku.config.IDKeyStrength))
	if err != nil {
		return nil, tracing.Trace(span, contracts.NewInternalError(err, "Failed to generate API Key."))
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
func (aku *apiKeyUtilsImpl) Parse(ctx context.Context, key string) (*domain.ParsedAPIKey, *contracts.APIError) {
	_, span := apiKeyUtilsTracer.Start(ctx, "utils.api_key.parse")
	defer span.End()

	parts := strings.SplitN(key, "_", 5)

	// We expect there to be parts to the key
	if len(parts) != 5 {
		return nil, aku.invalidAPIKeyError(span, key)
	}

	// The first two parts of the key should be the secret key prefix
	if !strings.HasPrefix(parts[0]+"_"+parts[1]+"_", string(domain.APIKeyPrefixSecretKey)) {
		return nil, aku.invalidAPIKeyError(span, key)
	}

	// The last part of the key should be at least 7 characters
	secretPlusChk := parts[4]
	if len(secretPlusChk) < 6 {
		return nil, aku.invalidAPIKeyError(span, key)
	}

	// Break down the key into the key elements
	id := parts[3]
	secret := secretPlusChk[:len(secretPlusChk)-6]
	chk := secretPlusChk[len(secretPlusChk)-6:]

	// If the checksum doesn't match, we know we didn't mint this key and we can fail fast
	expected := genKeyChecksum(id, secret)
	if expected != chk {
		return nil, aku.invalidAPIKeyError(span, key)
	}

	// The middle part of the key should be the account mode
	appMode := parts[2]
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
func (aku *apiKeyUtilsImpl) GenSecretHMAC(ctx context.Context, secret string) ([]byte, *contracts.APIError) {
	_, span := apiKeyUtilsTracer.Start(ctx, "utils.api_key.generate_secret_hmac")
	defer span.End()

	storedHMAC, err := hmacSHA256(aku.config.Pepper, []byte(secret))
	if err != nil {
		return nil, tracing.Trace(span, contracts.NewInternalError(err, "Failed to generate HMAC."))
	}

	return storedHMAC, nil
}

// VerifySecretHMAC verifies a given secret against a expected HMAC.
func (aku *apiKeyUtilsImpl) VerifySecretHMAC(ctx context.Context, secret string, expectedHMAC []byte) (bool, *contracts.APIError) {
	_, span := apiKeyUtilsTracer.Start(ctx, "utils.api_key.verify_secret_hmac")
	defer span.End()

	computedHMAC, err := hmacSHA256(aku.config.Pepper, []byte(secret))
	if err != nil {
		return false, tracing.Trace(span, contracts.NewInternalError(err, "Failed to generate HMAC for verification."))
	}

	return hmac.Equal(computedHMAC, expectedHMAC), nil
}

// SanitizeForDisplay sanitizes a given API key for display.
//
// Ex: aug_sk_prod_aG****UCZu
func (aku *apiKeyUtilsImpl) SanitizeForDisplay(apiKey string) string {
	return sanitize.SanitizeString(apiKey, len(domain.APIKeyPrefixSecretKey)+len(constants.AccountModeProduction)+3, 4)
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
	return 44
}

func hmacSHA256(key, data []byte) ([]byte, error) {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil), nil
}

func (aku *apiKeyUtilsImpl) invalidAPIKeyError(span trace.Span, key string) *contracts.APIError {
	message := fmt.Sprintf("%s: %s. Valid API keys start with '%s'.", ErrAPIKeyInvalid, aku.SanitizeForDisplay(key), string(domain.APIKeyPrefixSecretKey))
	return tracing.Trace(span, contracts.NewAuthenticationError(message))
}
