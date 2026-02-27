package apikey

import "time"

type DocAPIKey struct {
	ID              int64
	TypeID          string
	APIKeyID        string
	EncryptedSecret string
	CreatedAt       time.Time
	UpdatedAt       time.Time
	APIKeyExpiresAt *time.Time
	APIKeyRevokedAt *time.Time
}

// IsAPIKeyExpired reports whether the underlying API key has passed its expiration time.
func (m *DocAPIKey) IsAPIKeyExpired() bool {
	return m.APIKeyExpiresAt != nil && m.APIKeyExpiresAt.Before(time.Now().UTC())
}

// IsAPIKeyRevoked reports whether the underlying API key has been revoked.
func (m *DocAPIKey) IsAPIKeyRevoked() bool {
	return m.APIKeyRevokedAt != nil
}
