package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleNotificationPreferenceID = "nfpf_01h9z8q1w2e3r4t5y6u7nfpf"

// A per-(user, category) notification channel preference.
//
// A preference with a `null` category is the user's global default; a category-specific preference overrides it.
type NotificationPreference struct {
	// Preference ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=notification_preference"`
	// The notification category this preference applies to.
	//
	// `null` for the global default that applies to all categories without a specific preference.
	Category *string `json:"category"`
	// Whether in-app (bell) notifications are delivered for this category.
	InAppEnabled bool `json:"in_app_enabled"`
	// Whether email notifications are delivered for this category.
	EmailEnabled bool `json:"email_enabled"`
	// Whether push notifications are delivered for this category.
	PushEnabled bool `json:"push_enabled"`
	// How email delivery for this category is batched.
	//
	// - `instant`: send an email as soon as an eligible notification occurs.
	// - `hourly`: batch eligible notifications into a single hourly email.
	// - `daily`: batch eligible notifications into a single daily email.
	// - `off`: never send email for this category, even when email delivery is otherwise enabled.
	Digest constants.NotificationDigest `json:"digest" validate:"required"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last update timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleNotificationPreference = &NotificationPreference{
	ID:           SampleNotificationPreferenceID,
	Object:       constants.ObjectTypeNotificationPreference,
	Category:     new("chat.message"),
	InAppEnabled: true,
	EmailEnabled: true,
	PushEnabled:  false,
	Digest:       constants.NotificationDigestInstant,
	CreatedAt:    timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:    timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*NotificationPreference) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleNotificationPreference)
}
