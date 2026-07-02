package constants

// NotificationTargetType identifies what a notification send is aimed at. It is an enum (not a boolean) so new target kinds (groups, roles, segments, …) can be added without a breaking change to the send API.
type NotificationTargetType string

const (
	// NotificationTargetTypeAccountUser targets a single account user (a per-user notification).
	NotificationTargetTypeAccountUser NotificationTargetType = "account_user"
	// NotificationTargetTypeAccount targets every user in an account (a broadcast announcement).
	NotificationTargetTypeAccount NotificationTargetType = "account"
)

func (t NotificationTargetType) IsValid() bool {
	switch t {
	case NotificationTargetTypeAccountUser, NotificationTargetTypeAccount:
		return true
	default:
		return false
	}
}

func (t NotificationTargetType) EnumValues() []string {
	return []string{
		string(NotificationTargetTypeAccountUser),
		string(NotificationTargetTypeAccount),
	}
}

func (t *NotificationTargetType) StringPtr() *string {
	if t == nil {
		return nil
	}
	v := string(*t)
	return &v
}
