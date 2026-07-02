package domain

import "time"

// NotificationPreference is a per-(account_user, category) channel preference. A row with an empty category is the user's global default; a category-specific row overrides it.
type NotificationPreference struct {
	ID            string
	AccountID     string
	AccountUserID string
	Category      string
	InAppEnabled  bool
	EmailEnabled  bool
	PushEnabled   bool
	Digest        string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// UpsertNotificationPreferenceInput is the validated input for creating/replacing a preference row.
type UpsertNotificationPreferenceInput struct {
	Category     string
	InAppEnabled bool
	EmailEnabled bool
	PushEnabled  bool
	Digest       string
}

// EffectiveNotificationPreference is the resolved channel decision for a recipient + category, falling back to the channel defaults (in-app on; email/push off; digest off) when no row exists.
type EffectiveNotificationPreference struct {
	InAppEnabled bool
	EmailEnabled bool
	PushEnabled  bool
	Digest       string
}

// DefaultEffectiveNotificationPreference is applied when a recipient has no matching preference row.
// In-app (the bell) is on by default; email and push are off so chat messages do not email a user per message until they explicitly opt in by creating a preference row. The digest only governs email cadence, so it stays "off" alongside the disabled email channel.
func DefaultEffectiveNotificationPreference() EffectiveNotificationPreference {
	return EffectiveNotificationPreference{
		InAppEnabled: true,
		EmailEnabled: false,
		PushEnabled:  false,
		Digest:       string(EffectiveDigestOff),
	}
}

// EffectiveDigestOff is the digest applied when the email channel is off by default.
const EffectiveDigestOff = "off"
