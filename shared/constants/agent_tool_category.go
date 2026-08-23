package constants

// AgentToolCategory is where an agent tool's behavior comes from.
type AgentToolCategory string

const (
	// AgentToolCategoryBuiltIn is a capability implemented by the agent runtime itself.
	AgentToolCategoryBuiltIn AgentToolCategory = "built_in"
	// AgentToolCategoryAPIEndpoint is an operation of this API exposed as a tool.
	AgentToolCategoryAPIEndpoint AgentToolCategory = "api_endpoint"
)

func (c AgentToolCategory) IsValid() bool {
	switch c {
	case AgentToolCategoryBuiltIn, AgentToolCategoryAPIEndpoint:
		return true
	default:
		return false
	}
}

func (c AgentToolCategory) EnumValues() []string {
	return []string{
		string(AgentToolCategoryBuiltIn),
		string(AgentToolCategoryAPIEndpoint),
	}
}
