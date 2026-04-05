package constants

// AccountUserStatus represents the status of an account user.
type AccountUserStatus string

const (
	// AccountUserStatusActive indicates that the account user is active.
	AccountUserStatusActive AccountUserStatus = "active"
	// AccountUserStatusDisabled indicates that the account user is disabled (locked).
	AccountUserStatusDisabled AccountUserStatus = "disabled"
	// AccountUserStatusRemoved indicates that the account user has been soft-deleted.
	AccountUserStatusRemoved AccountUserStatus = "removed"
)

func (m AccountUserStatus) IsValid() bool {
	switch m {
	case AccountUserStatusActive, AccountUserStatusDisabled, AccountUserStatusRemoved:
		return true
	default:
		return false
	}
}

func (m AccountUserStatus) EnumValues() []string {
	return []string{string(AccountUserStatusActive), string(AccountUserStatusDisabled), string(AccountUserStatusRemoved)}
}
