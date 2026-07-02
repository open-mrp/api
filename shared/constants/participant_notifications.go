package constants

// ParticipantNotifications is a participant's notification preference for a conversation. It is an enum (not a boolean) so finer-grained levels (e.g. mentions-only, muted-until) can be added without a breaking change to the API.
type ParticipantNotifications string

const (
	// ParticipantNotificationsUnmuted means the participant receives normal notifications.
	ParticipantNotificationsUnmuted ParticipantNotifications = "unmuted"
	// ParticipantNotificationsMuted means the participant has muted the conversation.
	ParticipantNotificationsMuted ParticipantNotifications = "muted"
)

func (s ParticipantNotifications) IsValid() bool {
	switch s {
	case ParticipantNotificationsUnmuted, ParticipantNotificationsMuted:
		return true
	default:
		return false
	}
}

func (s ParticipantNotifications) EnumValues() []string {
	return []string{string(ParticipantNotificationsUnmuted), string(ParticipantNotificationsMuted)}
}

func (s *ParticipantNotifications) StringPtr() *string {
	if s == nil {
		return nil
	}
	v := string(*s)
	return &v
}

// ParticipantNotificationsFromMuted maps the persisted boolean to its notifications enum.
func ParticipantNotificationsFromMuted(muted bool) ParticipantNotifications {
	if muted {
		return ParticipantNotificationsMuted
	}
	return ParticipantNotificationsUnmuted
}
