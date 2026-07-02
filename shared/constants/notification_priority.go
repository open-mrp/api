package constants

// NotificationPriority represents the delivery priority of an in-app notification.
type NotificationPriority string

const (
	// NotificationPriorityLow indicates a low-priority, informational notification.
	NotificationPriorityLow NotificationPriority = "low"
	// NotificationPriorityNormal indicates a standard-priority notification (default).
	NotificationPriorityNormal NotificationPriority = "normal"
	// NotificationPriorityHigh indicates a high-priority notification that should stand out.
	NotificationPriorityHigh NotificationPriority = "high"
	// NotificationPriorityUrgent indicates an urgent notification requiring prompt attention.
	NotificationPriorityUrgent NotificationPriority = "urgent"
)

func (p NotificationPriority) IsValid() bool {
	switch p {
	case NotificationPriorityLow, NotificationPriorityNormal, NotificationPriorityHigh, NotificationPriorityUrgent:
		return true
	default:
		return false
	}
}

func (p NotificationPriority) EnumValues() []string {
	return []string{string(NotificationPriorityLow), string(NotificationPriorityNormal), string(NotificationPriorityHigh), string(NotificationPriorityUrgent)}
}

func (p *NotificationPriority) StringPtr() *string {
	if p == nil {
		return nil
	}
	v := string(*p)
	return &v
}
