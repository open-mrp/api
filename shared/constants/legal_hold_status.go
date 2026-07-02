package constants

// LegalHoldStatus is whether a conversation is under legal hold, which exempts it from automatic retention purging and from GDPR redaction until the hold is released. It is an enum (not a boolean) so additional states (e.g. pending-review) can be added without a breaking change.
type LegalHoldStatus string

const (
	// LegalHoldStatusReleased means the conversation is not under legal hold (the default).
	LegalHoldStatusReleased LegalHoldStatus = "released"
	// LegalHoldStatusHeld means the conversation is under legal hold and exempt from purge/redaction.
	LegalHoldStatusHeld LegalHoldStatus = "held"
)

func (s LegalHoldStatus) IsValid() bool {
	switch s {
	case LegalHoldStatusReleased, LegalHoldStatusHeld:
		return true
	default:
		return false
	}
}

func (s LegalHoldStatus) EnumValues() []string {
	return []string{string(LegalHoldStatusReleased), string(LegalHoldStatusHeld)}
}

func (s *LegalHoldStatus) StringPtr() *string {
	if s == nil {
		return nil
	}
	v := string(*s)
	return &v
}

// LegalHoldStatusFromHeld maps the persisted boolean to its legal-hold-status enum.
func LegalHoldStatusFromHeld(held bool) LegalHoldStatus {
	if held {
		return LegalHoldStatusHeld
	}
	return LegalHoldStatusReleased
}
