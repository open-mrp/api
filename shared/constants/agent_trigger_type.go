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
	// AgentTriggerTypeChat indicates that the agent run is initiated by a chat message (the run is linked to a conversation and posts its reply back into it).
	AgentTriggerTypeChat AgentTriggerType = "chat"
)

func (m AgentTriggerType) IsValid() bool {
	switch m {
	case AgentTriggerTypeScheduled, AgentTriggerTypeManual, AgentTriggerTypeEvent, AgentTriggerTypeChat:
		return true
	default:
		return false
	}
}

func (m AgentTriggerType) EnumValues() []string {
	return []string{string(AgentTriggerTypeScheduled), string(AgentTriggerTypeManual), string(AgentTriggerTypeEvent), string(AgentTriggerTypeChat)}
}
