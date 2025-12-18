package testutil

import (
	"time"

	"github.com/augno/api/services/auth-service/internal/domain"
)

var (
	apiKeyExpiresAt        = time.Now().UTC().Add(time.Hour * 24 * 30)
	apiKeyExpiresAtExpired = time.Now().UTC().Add(-time.Hour * 24 * 30)
)

func GetValidTestAPIKeyModel(secretHash []byte) *domain.APIKey {
	return &domain.APIKey{
		ID:             EntityIDAPIKeyValidSandboxMode,
		LastFour:       "1234",
		OwnerAccountID: EntityIDAccount,
		RoleID:         EntityIDRole,
		ExpiresAt:      &apiKeyExpiresAt,
		SecretHash:     secretHash,
	}
}

func GetValidProdAPIKeyModel(secretHash []byte) *domain.APIKey {
	return &domain.APIKey{
		ID:             EntityIDAPIKeyValidProdMode,
		LastFour:       "1234",
		OwnerAccountID: EntityIDAccount,
		RoleID:         EntityIDRole,
		ExpiresAt:      &apiKeyExpiresAt,
		SecretHash:     secretHash,
	}
}

func GetExpiredAPIKeyModel(secretHash []byte) *domain.APIKey {
	return &domain.APIKey{
		ID:             EntityIDAPIKeyExpired,
		LastFour:       "1234",
		OwnerAccountID: EntityIDAccount,
		RoleID:         EntityIDRole,
		ExpiresAt:      &apiKeyExpiresAtExpired,
		SecretHash:     secretHash,
	}
}

func GetBadSecretAPIKeyModel(secretHash []byte) *domain.APIKey {
	return &domain.APIKey{
		ID:             EntityIDAPIKeyBadSecret,
		LastFour:       "1234",
		OwnerAccountID: EntityIDAccount,
		RoleID:         EntityIDRole,
		ExpiresAt:      &apiKeyExpiresAt,
		SecretHash:     secretHash,
	}
}

func GetNeverExpiresAPIKeyModel(secretHash []byte) *domain.APIKey {
	return &domain.APIKey{
		ID:             EntityIDAPIKeyNeverExpires,
		LastFour:       "4HAj",
		OwnerAccountID: EntityIDAccount,
		RoleID:         EntityIDRole,
		ExpiresAt:      nil,
		SecretHash:     secretHash,
	}
}

type APIKeySvcTestModels struct {
	ValidTestModel *domain.APIKey
	ValidProdModel *domain.APIKey
	ExpiredModel   *domain.APIKey
	BadSecretModel *domain.APIKey
}

func NewAPIKeySvcTestModels(apiKeyUtils interface{}) *APIKeySvcTestModels {
	return &APIKeySvcTestModels{}
}

func (m *APIKeySvcTestModels) SetValidTestModel(secretHash []byte) {
	m.ValidTestModel = GetValidTestAPIKeyModel(secretHash)
}

func (m *APIKeySvcTestModels) SetValidProdModel(secretHash []byte) {
	m.ValidProdModel = GetValidProdAPIKeyModel(secretHash)
}

func (m *APIKeySvcTestModels) SetExpiredModel(secretHash []byte) {
	m.ExpiredModel = GetExpiredAPIKeyModel(secretHash)
}

func (m *APIKeySvcTestModels) SetBadSecretModel(secretHash []byte) {
	m.BadSecretModel = GetBadSecretAPIKeyModel(secretHash)
}
