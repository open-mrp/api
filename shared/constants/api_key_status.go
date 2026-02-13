package constants

// APIKeyStatus represents the status of an API key.
type APIKeyStatus string

const (
	// APIKeyStatusActive indicates that the API key is active and can be used to authenticate requests.
	APIKeyStatusActive APIKeyStatus = "active"
	// APIKeyStatusExpired indicates that the API key has expired and can no longer be used to authenticate requests.
	APIKeyStatusExpired APIKeyStatus = "expired"
	// APIKeyStatusRevoked indicates that the API key has been revoked and can no longer be used to authenticate requests.
	APIKeyStatusRevoked APIKeyStatus = "revoked"
)

func (s APIKeyStatus) IsValid() bool {
	switch s {
	case APIKeyStatusActive, APIKeyStatusExpired, APIKeyStatusRevoked:
		return true
	default:
		return false
	}
}

func (s APIKeyStatus) EnumValues() []string {
	return []string{string(APIKeyStatusActive), string(APIKeyStatusExpired), string(APIKeyStatusRevoked)}
}
