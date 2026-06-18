package constants

// AccountIntegrationStatus represents the lifecycle status of an account integration.
type AccountIntegrationStatus string

const (
	// AccountIntegrationStatusActive indicates the integration is active and available for use.
	AccountIntegrationStatusActive AccountIntegrationStatus = "active"
	// AccountIntegrationStatusInactive indicates the integration is deactivated; its stored credentials are retained but it cannot be used.
	AccountIntegrationStatusInactive AccountIntegrationStatus = "inactive"
)

func (m AccountIntegrationStatus) IsValid() bool {
	switch m {
	case AccountIntegrationStatusActive, AccountIntegrationStatusInactive:
		return true
	default:
		return false
	}
}

func (m AccountIntegrationStatus) EnumValues() []string {
	return []string{string(AccountIntegrationStatusActive), string(AccountIntegrationStatusInactive)}
}

// AccountIntegrationStatusFromActive maps the stored is_active boolean to its public status value.
func AccountIntegrationStatusFromActive(active bool) AccountIntegrationStatus {
	if active {
		return AccountIntegrationStatusActive
	}
	return AccountIntegrationStatusInactive
}
