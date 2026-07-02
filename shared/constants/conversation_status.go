package constants

// ConversationStatus is the caller's effective view of a conversation. It is an enum (not a boolean) so new states (e.g. frozen, deleted) can be added without a breaking change to the
// API. Hidden is per-caller and takes precedence over the account-level archived state.
type ConversationStatus string

const (
	// ConversationStatusActive is a normal, visible conversation.
	ConversationStatusActive ConversationStatus = "active"
	// ConversationStatusArchived is archived at the account level.
	ConversationStatusArchived ConversationStatus = "archived"
	// ConversationStatusHidden is hidden from the caller's list (per-caller, dismissed).
	ConversationStatusHidden ConversationStatus = "hidden"
)

func (s ConversationStatus) IsValid() bool {
	switch s {
	case ConversationStatusActive, ConversationStatusArchived, ConversationStatusHidden:
		return true
	default:
		return false
	}
}

func (s ConversationStatus) EnumValues() []string {
	return []string{string(ConversationStatusActive), string(ConversationStatusArchived), string(ConversationStatusHidden)}
}

func (s *ConversationStatus) StringPtr() *string {
	if s == nil {
		return nil
	}
	v := string(*s)
	return &v
}
