package constants

// AgentDefinitionType classifies an agent definition as system-provided or user-created.
type AgentDefinitionType string

const (
	// AgentDefinitionTypeSystem indicates that the agent definition is provided by the system.
	AgentDefinitionTypeSystem AgentDefinitionType = "system"
	// AgentDefinitionTypeCustom indicates that the agent definition is created by the user.
	AgentDefinitionTypeCustom AgentDefinitionType = "custom"
)

func (m AgentDefinitionType) IsValid() bool {
	switch m {
	case AgentDefinitionTypeSystem, AgentDefinitionTypeCustom:
		return true
	default:
		return false
	}
}

func (m AgentDefinitionType) EnumValues() []string {
	return []string{string(AgentDefinitionTypeSystem), string(AgentDefinitionTypeCustom)}
}
