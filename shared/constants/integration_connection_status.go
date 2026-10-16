package constants

// IntegrationConnectionStatus is whether the last call OpenMRP made to an integration's provider succeeded.
type IntegrationConnectionStatus string

const (
	// IntegrationConnectionStatusConnected means the provider accepted the last call.
	IntegrationConnectionStatusConnected IntegrationConnectionStatus = "connected"
	// IntegrationConnectionStatusInvalidCredentials means the provider rejected the stored credentials.
	IntegrationConnectionStatusInvalidCredentials IntegrationConnectionStatus = "invalid_credentials"
	// IntegrationConnectionStatusUnknown means no call has settled it yet.
	IntegrationConnectionStatusUnknown IntegrationConnectionStatus = "unknown"
)

func (s IntegrationConnectionStatus) IsValid() bool {
	switch s {
	case IntegrationConnectionStatusConnected, IntegrationConnectionStatusInvalidCredentials, IntegrationConnectionStatusUnknown:
		return true
	default:
		return false
	}
}

func (s IntegrationConnectionStatus) EnumValues() []string {
	return []string{string(IntegrationConnectionStatusConnected), string(IntegrationConnectionStatusInvalidCredentials), string(IntegrationConnectionStatusUnknown)}
}
