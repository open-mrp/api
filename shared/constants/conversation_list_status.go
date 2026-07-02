package constants

// ConversationListStatus filters the caller's conversation list by visibility.
type ConversationListStatus string

const (
	// ConversationListStatusActive returns visible conversations (not hidden by the caller).
	ConversationListStatusActive ConversationListStatus = "active"
	// ConversationListStatusHidden returns conversations the caller has hidden.
	ConversationListStatusHidden ConversationListStatus = "hidden"
)

func (s ConversationListStatus) IsValid() bool {
	switch s {
	case ConversationListStatusActive, ConversationListStatusHidden:
		return true
	default:
		return false
	}
}

func (s ConversationListStatus) EnumValues() []string {
	return []string{string(ConversationListStatusActive), string(ConversationListStatusHidden)}
}

func (s *ConversationListStatus) StringPtr() *string {
	if s == nil {
		return nil
	}
	v := string(*s)
	return &v
}
