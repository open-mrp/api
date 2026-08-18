package constants

// AgentMemoryCategory is the kind of information an agent memory holds, used to group related memories.
type AgentMemoryCategory string

const (
	// AgentMemoryCategoryPreference is how someone likes things done, such as a customer who always wants express shipping.
	AgentMemoryCategoryPreference AgentMemoryCategory = "preference"
	// AgentMemoryCategoryFact is a durable detail worth remembering about the account or one of its records, such as a customer's typical order size.
	AgentMemoryCategoryFact AgentMemoryCategory = "fact"
	// AgentMemoryCategoryInstruction is standing guidance for agents to follow, such as always confirming freight before issuing an order.
	AgentMemoryCategoryInstruction AgentMemoryCategory = "instruction"
)

func (c AgentMemoryCategory) IsValid() bool {
	switch c {
	case AgentMemoryCategoryPreference, AgentMemoryCategoryFact, AgentMemoryCategoryInstruction:
		return true
	default:
		return false
	}
}

func (c AgentMemoryCategory) EnumValues() []string {
	return []string{string(AgentMemoryCategoryPreference), string(AgentMemoryCategoryFact), string(AgentMemoryCategoryInstruction)}
}

func (c *AgentMemoryCategory) StringPtr() *string {
	if c == nil {
		return nil
	}
	v := string(*c)
	return &v
}
