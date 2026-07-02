package constants

// ConversationType is the kind of conversation container.
type ConversationType string

const (
	// ConversationTypeDM is a 1:1 direct message between exactly two users.
	ConversationTypeDM ConversationType = "direct_message"
	// ConversationTypeGroup is a named conversation with 2+ user/agent participants.
	ConversationTypeGroup ConversationType = "group"
	// ConversationTypeSystem is a per-account/per-category system channel for alerts.
	ConversationTypeSystem ConversationType = "system"
)

func (t ConversationType) IsValid() bool {
	switch t {
	case ConversationTypeDM, ConversationTypeGroup, ConversationTypeSystem:
		return true
	default:
		return false
	}
}

func (t ConversationType) EnumValues() []string {
	return []string{
		string(ConversationTypeDM),
		string(ConversationTypeGroup),
		string(ConversationTypeSystem),
	}
}

func (t *ConversationType) StringPtr() *string {
	if t == nil {
		return nil
	}
	v := string(*t)
	return &v
}
