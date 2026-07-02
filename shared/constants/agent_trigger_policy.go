package constants

// AgentTriggerPolicy controls when an agent participant is invoked in response to a human message.
// It is an enum so new policies can be added without a breaking change to the API.
type AgentTriggerPolicy string

const (
	// AgentTriggerPolicyMention fires only when the agent is @mentioned (body contains "@" + a handle from the participant's trigger keywords).
	AgentTriggerPolicyMention AgentTriggerPolicy = "mention"
	// AgentTriggerPolicyKeyword fires when the body contains any of the participant's trigger keywords.
	AgentTriggerPolicyKeyword AgentTriggerPolicy = "keyword"
	// AgentTriggerPolicyAlways fires on every human message in the conversation.
	AgentTriggerPolicyAlways AgentTriggerPolicy = "always"
)

func (p AgentTriggerPolicy) IsValid() bool {
	switch p {
	case AgentTriggerPolicyMention, AgentTriggerPolicyKeyword, AgentTriggerPolicyAlways:
		return true
	default:
		return false
	}
}

func (p AgentTriggerPolicy) EnumValues() []string {
	return []string{
		string(AgentTriggerPolicyMention),
		string(AgentTriggerPolicyKeyword),
		string(AgentTriggerPolicyAlways),
	}
}

func (p *AgentTriggerPolicy) StringPtr() *string {
	if p == nil {
		return nil
	}
	v := string(*p)
	return &v
}
