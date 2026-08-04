package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleNotificationPreferenceID = "nfpf_thr6wg569txs"

// One user's choice of which channels a category of notification is delivered on.
//
// Preferences belong to the user's membership in a single account, so the same person can be notified differently in each account they belong to. A preference with no category is that user's global default, and a category-specific preference overrides it. Where neither exists, in-app notifications are delivered and email and push are not.
//
// Chat notifications are the only ones these settings currently govern: notifications in every other category reach the in-app feed and are never emailed, whatever is stored here.
type NotificationPreference struct {
	// Preference ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=notification_preference"`
	// The notification category this preference applies to.
	//
	// A preference with no category is the user's global default, used for every category they have not set a specific preference for.
	Category *string `json:"category"`
	// Whether notifications in this category appear in the user's in-app feed.
	//
	// A direct @mention is always delivered in-app, even when this is disabled.
	InAppEnabled bool `json:"in_app_enabled"`
	// Whether notifications in this category are also emailed to the user.
	//
	// Email is additionally suppressed for a conversation the user has muted, and only sent on the cadence set by `digest`.
	EmailEnabled bool `json:"email_enabled"`
	// Whether notifications in this category are also sent as push notifications.
	//
	// Push delivery is not available yet; the choice is stored for when it is.
	PushEnabled bool `json:"push_enabled"`
	// How often email for this category is sent.
	//
	// - `instant`: send an email as soon as an eligible notification occurs.
	// - `hourly`: collect eligible notifications into a single hourly email.
	// - `daily`: collect eligible notifications into a single daily email.
	// - `off`: never send email for this category, even when email is otherwise enabled.
	//
	// This governs email only; in-app delivery is unaffected. Batched sending is not running yet, so `hourly` and `daily` currently hold email back in the same way as `off`.
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
