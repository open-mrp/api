package constants

// ConversationAudience is the direction of a conversation, orthogonal to its type. "internal" is a team-only conversation (the customer is never a participant — e.g. a DM, group, or object-linked discussion ABOUT a customer); "customer" is an external customer-facing case the customer participates in and sees (portal support or an email-bridged thread). It drives the support inbox and the per-message customer-visible read filtering.
type ConversationAudience string

const (
	// ConversationAudienceInternal is a team-only conversation.
	ConversationAudienceInternal ConversationAudience = "internal"
	// ConversationAudienceCustomer is an external customer-facing case.
	ConversationAudienceCustomer ConversationAudience = "customer"
)

func (a ConversationAudience) IsValid() bool {
	switch a {
	case ConversationAudienceInternal, ConversationAudienceCustomer:
		return true
	default:
		return false
	}
}

func (a ConversationAudience) EnumValues() []string {
	return []string{string(ConversationAudienceInternal), string(ConversationAudienceCustomer)}
}

func (a *ConversationAudience) StringPtr() *string {
	if a == nil {
		return nil
	}
	v := string(*a)
	return &v
}
