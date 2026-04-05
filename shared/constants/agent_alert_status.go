package constants

// AgentAlertStatus represents the status of an agent alert.
type AgentAlertStatus string

const (
	// AgentAlertStatusOpen indicates the alert has not yet been acknowledged.
	AgentAlertStatusOpen AgentAlertStatus = "open"
	// AgentAlertStatusAcknowledged indicates the alert has been seen and acknowledged by a user.
	AgentAlertStatusAcknowledged AgentAlertStatus = "acknowledged"
)

func (s AgentAlertStatus) IsValid() bool {
	switch s {
	case AgentAlertStatusOpen, AgentAlertStatusAcknowledged:
		return true
	default:
		return false
	}
}

func (s AgentAlertStatus) EnumValues() []string {
	return []string{string(AgentAlertStatusOpen), string(AgentAlertStatusAcknowledged)}
}
