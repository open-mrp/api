package constants

// NotificationDigest controls how email delivery for a notification category is batched. It is an enum (not a boolean) so new cadences can be added without a breaking change to the API.
type NotificationDigest string

const (
	// NotificationDigestInstant delivers an email per eligible notification immediately.
	NotificationDigestInstant NotificationDigest = "instant"
	// NotificationDigestHourly batches eligible notifications into an hourly email.
	NotificationDigestHourly NotificationDigest = "hourly"
	// NotificationDigestDaily batches eligible notifications into a daily email.
	NotificationDigestDaily NotificationDigest = "daily"
	// NotificationDigestOff disables email delivery for the category.
	NotificationDigestOff NotificationDigest = "off"
)

func (d NotificationDigest) IsValid() bool {
	switch d {
	case NotificationDigestInstant, NotificationDigestHourly, NotificationDigestDaily, NotificationDigestOff:
		return true
	default:
		return false
	}
}

func (d NotificationDigest) EnumValues() []string {
	return []string{
		string(NotificationDigestInstant),
		string(NotificationDigestHourly),
		string(NotificationDigestDaily),
		string(NotificationDigestOff),
	}
}

func (d *NotificationDigest) StringPtr() *string {
	if d == nil {
		return nil
	}
	s := string(*d)
	return &s
}
