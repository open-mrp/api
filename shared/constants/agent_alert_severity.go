package constants

// AgentAlertSeverity represents the severity level of an agent alert.
type AgentAlertSeverity string

const (
	// AgentAlertSeverityInfo indicates an informational alert that requires no immediate action.
	AgentAlertSeverityInfo AgentAlertSeverity = "info"
	// AgentAlertSeverityWarning indicates a potential issue that should be reviewed.
	AgentAlertSeverityWarning AgentAlertSeverity = "warning"
	// AgentAlertSeverityUrgent indicates an issue that requires prompt attention.
	AgentAlertSeverityUrgent AgentAlertSeverity = "urgent"
	// AgentAlertSeverityCritical indicates a severe issue requiring immediate action.
	AgentAlertSeverityCritical AgentAlertSeverity = "critical"
)

func (s AgentAlertSeverity) IsValid() bool {
	switch s {
	case AgentAlertSeverityInfo, AgentAlertSeverityWarning, AgentAlertSeverityUrgent, AgentAlertSeverityCritical:
		return true
	default:
		return false
	}
}

func (s AgentAlertSeverity) EnumValues() []string {
	return []string{string(AgentAlertSeverityInfo), string(AgentAlertSeverityWarning), string(AgentAlertSeverityUrgent), string(AgentAlertSeverityCritical)}
}
