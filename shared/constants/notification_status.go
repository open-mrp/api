package constants

// NotificationStatus is the lifecycle state of an in-app notification. Modeled as a constant (rather than seen/read booleans) so new states can be added without breaking existing clients. The state is derived from the seen/read/dismissed timestamps.
type NotificationStatus string

const (
	// NotificationStatusUnseen indicates the notification has not yet appeared in the dropdown.
	NotificationStatusUnseen NotificationStatus = "unseen"
	// NotificationStatusSeen indicates the notification has been surfaced but not opened.
	NotificationStatusSeen NotificationStatus = "seen"
	// NotificationStatusRead indicates the notification has been explicitly opened.
	NotificationStatusRead NotificationStatus = "read"
	// NotificationStatusDismissed indicates the notification has been dismissed.
	NotificationStatusDismissed NotificationStatus = "dismissed"
)

func (s NotificationStatus) IsValid() bool {
	switch s {
	case NotificationStatusUnseen, NotificationStatusSeen, NotificationStatusRead, NotificationStatusDismissed:
		return true
	default:
		return false
	}
}

func (s NotificationStatus) EnumValues() []string {
	return []string{string(NotificationStatusUnseen), string(NotificationStatusSeen), string(NotificationStatusRead), string(NotificationStatusDismissed)}
}

func (s *NotificationStatus) StringPtr() *string {
	if s == nil {
		return nil
	}
	v := string(*s)
	return &v
}
