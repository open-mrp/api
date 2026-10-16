package constants

// NotificationChannel is where a notification is delivered.
type NotificationChannel string

const (
	// NotificationChannelInApp delivers to the in-app notification feed.
	NotificationChannelInApp NotificationChannel = "in_app"
	// NotificationChannelEmail delivers by email.
	NotificationChannelEmail NotificationChannel = "email"
	// NotificationChannelPush delivers as a push notification.
	NotificationChannelPush NotificationChannel = "push"
)

func (c NotificationChannel) IsValid() bool {
	switch c {
	case NotificationChannelInApp, NotificationChannelEmail, NotificationChannelPush:
		return true
	}
	return false
}

func (c NotificationChannel) EnumValues() []string {
	return []string{string(NotificationChannelInApp), string(NotificationChannelEmail), string(NotificationChannelPush)}
}

func (v *NotificationChannel) StringPtr() *string {
	if v == nil {
		return nil
	}
	s := string(*v)
	return &s
}
