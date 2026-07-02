package constants

// AnnouncementScope is the reach of an announcement: a single account's users, or every user on the platform.
type AnnouncementScope string

const (
	// AnnouncementScopeAccount targets all active users within one account.
	AnnouncementScopeAccount AnnouncementScope = "account"
	// AnnouncementScopePlatform targets every user across all accounts (platform-wide).
	AnnouncementScopePlatform AnnouncementScope = "platform"
)

func (s AnnouncementScope) IsValid() bool {
	switch s {
	case AnnouncementScopeAccount, AnnouncementScopePlatform:
		return true
	default:
		return false
	}
}

func (s AnnouncementScope) EnumValues() []string {
	return []string{
		string(AnnouncementScopeAccount),
		string(AnnouncementScopePlatform),
	}
}

func (s *AnnouncementScope) StringPtr() *string {
	if s == nil {
		return nil
	}
	v := string(*s)
	return &v
}
