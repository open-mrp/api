package constants

// AgentAccountStatus describes the per-account activation status of an agent definition.
type AgentAccountStatus string

const (
	// AgentAccountStatusActive indicates the agent is active for this account.
	AgentAccountStatusActive AgentAccountStatus = "active"
	// AgentAccountStatusInactive indicates the agent is inactive for this account.
	AgentAccountStatusInactive AgentAccountStatus = "inactive"
)

func (m AgentAccountStatus) IsValid() bool {
	switch m {
	case AgentAccountStatusActive, AgentAccountStatusInactive:
		return true
	default:
		return false
	}
}

func (m AgentAccountStatus) EnumValues() []string {
	return []string{string(AgentAccountStatusActive), string(AgentAccountStatusInactive)}
}
