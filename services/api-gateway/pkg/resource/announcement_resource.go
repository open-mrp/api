package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleAnnouncementID = "an_01c4d5e6f7a8b9c0d1e2f3a4"

// A broadcast announcement shown in the bell feed, with the caller's per-user read state.
type Announcement struct {
	// Announcement ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=announcement"`
	// Reach of the announcement.
	//
	// - `account`: shown only to users within this account.
	// - `platform`: shown to every user across all accounts.
	Scope constants.AnnouncementScope `json:"scope" validate:"required"`
	// Category of the announcement.
	Category constants.NotificationCategory `json:"category" validate:"required"`
	// Human-readable title.
	Title string `json:"title" validate:"required"`
	// Preview/body text.
	Body *string `json:"body"`
	// Lifecycle status of the announcement for the calling actor, derived from their seen/read/dismissed receipt.
	//
	// - `unseen`: not yet surfaced in the caller's feed.
	// - `seen`: surfaced in the feed but not yet opened.
	// - `read`: opened by the caller.
	// - `dismissed`: dismissed by the caller.
	Status constants.NotificationStatus `json:"status" validate:"required"`
	// Delivery priority.
	Priority constants.NotificationPriority `json:"priority" validate:"required"`
	// The app resource this announcement links to.
	Resource *Entity `json:"resource" expandable:"true"`
	// When the announcement becomes visible in the feed.
	PublishAt time.Time `json:"publish_at" validate:"required"`
	// When the announcement stops being shown.
	ExpiresAt *time.Time `json:"expires_at"`
	// When the calling actor first saw the announcement.
	SeenAt *time.Time `json:"seen_at"`
	// When the calling actor opened the announcement.
	ReadAt *time.Time `json:"read_at"`
	// When the calling actor dismissed the announcement.
	DismissedAt *time.Time `json:"dismissed_at"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last update timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleAnnouncement = &Announcement{
	ID:        SampleAnnouncementID,
	Object:    constants.ObjectTypeAnnouncement,
	Scope:     constants.AnnouncementScopeAccount,
	Category:  constants.NotificationCategoryOrderUpdated,
	Status:    constants.NotificationStatusUnseen,
	Title:     "Scheduled maintenance",
	Body:      new("The platform will be briefly unavailable tonight at 2am UTC."),
	Priority:  constants.NotificationPriorityNormal,
	Resource:  NewEntity(SampleSalesOrderID, constants.ObjectTypeSalesOrder, new("Order #1042"), nil),
	PublishAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	ExpiresAt: timeutil.TimestampToTimePtr(sampleExpiresAtTimestamp),
	CreatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt: timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*Announcement) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleAnnouncement)
}
