package constants

// AgentTriggerType describes how an agent run is initiated.
type AgentTriggerType string

const (
	// AgentTriggerTypeScheduled indicates that the agent run is initiated by a scheduled event.
	AgentTriggerTypeScheduled AgentTriggerType = "scheduled"
	// AgentTriggerTypeManual indicates that the agent run is initiated by a manual event.
	AgentTriggerTypeManual AgentTriggerType = "manual"
	// AgentTriggerTypeEvent indicates that the agent run is initiated by an event.
	AgentTriggerTypeEvent AgentTriggerType = "event"
)

func (m AgentTriggerType) IsValid() bool {
	switch m {
	case AgentTriggerTypeScheduled, AgentTriggerTypeManual, AgentTriggerTypeEvent:
		return true
	default:
		return false
	}
}

func (m AgentTriggerType) EnumValues() []string {
	return []string{string(AgentTriggerTypeScheduled), string(AgentTriggerTypeManual), string(AgentTriggerTypeEvent)}
}
