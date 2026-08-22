package testutil

import (
	"time"

	"github.com/open-mrp/api/services/auth-service/internal/apikey"
)

var (
	apiKeyExpiresAt        = time.Now().UTC().Add(time.Hour * 24 * 30)
	apiKeyExpiresAtExpired = time.Now().UTC().Add(-time.Hour * 24 * 30)
)

func GetValidTestAPIKeyModel(secretHash []byte) *apikey.APIKey {
	return &apikey.APIKey{
		ID:             1,
		TypeID:         "apikey_sandbox",
		KeyID:          EntityIDAPIKeyValidSandboxMode,
		RedactedValue:  "mrp_sk_test_****1234",
		OwnerAccountID: EntityIDAccount,
		RoleID:         EntityIDRole,
		ExpiresAt:      &apiKeyExpiresAt,
		SecretHash:     secretHash,
	}
}

func GetValidProdAPIKeyModel(secretHash []byte) *apikey.APIKey {
	return &apikey.APIKey{
		ID:             2,
		TypeID:         "apikey_prod",
		KeyID:          EntityIDAPIKeyValidProdMode,
		RedactedValue:  "mrp_sk_prod_****1234",
		OwnerAccountID: EntityIDAccount,
		RoleID:         EntityIDRole,
		ExpiresAt:      &apiKeyExpiresAt,
		SecretHash:     secretHash,
	}
}

func GetExpiredAPIKeyModel(secretHash []byte) *apikey.APIKey {
	return &apikey.APIKey{
		ID:             3,
		TypeID:         "apikey_prod",
		KeyID:          EntityIDAPIKeyExpired,
		RedactedValue:  "mrp_sk_prod_****1234",
		OwnerAccountID: EntityIDAccount,
		RoleID:         EntityIDRole,
		ExpiresAt:      &apiKeyExpiresAtExpired,
		SecretHash:     secretHash,
	}
}

func GetBadSecretAPIKeyModel(secretHash []byte) *apikey.APIKey {
	return &apikey.APIKey{
		ID:             4,
		TypeID:         "apikey_prod",
		KeyID:          EntityIDAPIKeyBadSecret,
		RedactedValue:  "mrp_sk_prod_****1234",
		OwnerAccountID: EntityIDAccount,
		RoleID:         EntityIDRole,
		ExpiresAt:      &apiKeyExpiresAt,
		SecretHash:     secretHash,
	}
}

func GetNeverExpiresAPIKeyModel(secretHash []byte) *apikey.APIKey {
	return &apikey.APIKey{
		ID:             5,
		TypeID:         "apikey_prod",
		KeyID:          EntityIDAPIKeyNeverExpires,
		RedactedValue:  "mrp_sk_prod_****4HAj",
		OwnerAccountID: EntityIDAccount,
		RoleID:         EntityIDRole,
		ExpiresAt:      nil,
		SecretHash:     secretHash,
	}
}

type APIKeySvcTestModels struct {
	ValidTestModel *apikey.APIKey
	ValidProdModel *apikey.APIKey
	ExpiredModel   *apikey.APIKey
	BadSecretModel *apikey.APIKey
}
