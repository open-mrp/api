package constants

// ConversationAssigneeType names the kind of owner a customer-service case is assigned to.
type ConversationAssigneeType string

const (
	// ConversationAssigneeTypeAccountUser gives the case to an individual teammate.
	ConversationAssigneeTypeAccountUser ConversationAssigneeType = "account_user"
	// ConversationAssigneeTypeAccountGroup gives the case to a team, so anyone on it can pick it up.
	ConversationAssigneeTypeAccountGroup ConversationAssigneeType = "account_group"
)

func (t ConversationAssigneeType) IsValid() bool {
	switch t {
	case ConversationAssigneeTypeAccountUser, ConversationAssigneeTypeAccountGroup:
		return true
	default:
		return false
	}
}

func (t ConversationAssigneeType) EnumValues() []string {
	return []string{
		string(ConversationAssigneeTypeAccountUser),
		string(ConversationAssigneeTypeAccountGroup),
	}
}
