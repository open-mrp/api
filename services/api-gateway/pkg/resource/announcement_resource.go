package apiresource

import (
	"time"

	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/timeutil"
)

const SampleAnnouncementID = "an_m4vwgn2t8cqs"

// A broadcast announcement shown in the notification (bell) feed, carrying the calling user's own read state.
//
// A single announcement is published to everyone in an account, or to every user on the platform, and each user keeps their own seen, read, and dismissed state for it. The status and timestamps you read are therefore always the caller's, and never reflect what anyone else has done with the same announcement. Notifications addressed to one user are a separate resource.
type Announcement struct {
	// Announcement ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=announcement"`
	// Who the announcement reaches.
	//
	// - `account`: published to a single account and shown only to that account's users.
	// - `platform`: published by OpenMRP and shown to every user across all accounts.
	Scope constants.AnnouncementScope `json:"scope" validate:"required"`
	// The kind of event the announcement is about.
	//
	// Announcements draw on the same categories as notifications, such as `system.broadcast` or `order.updated`, and the category is chosen by whoever publishes the announcement. The set is open-ended and may grow over time, so clients should tolerate values they do not recognize.
	Category constants.NotificationCategory `json:"category" validate:"required"`
	// Short headline shown in the feed.
	Title string `json:"title" validate:"required"`
	// Supporting detail shown beneath the title.
	Body *string `json:"body"`
	// Where the announcement is in its lifecycle for the calling user.
	//
	// - `unseen`: not yet surfaced to the caller.
	// - `seen`: surfaced in the caller's feed but not opened.
	// - `read`: explicitly opened by the caller.
	// - `dismissed`: removed from the caller's feed.
	//
	// The status is derived from the caller's own seen, read, and dismissed timestamps and only ever moves forward, so the same announcement can show a different status for each user in the account.
	Status constants.NotificationStatus `json:"status" validate:"required"`
	// How prominently the announcement should be surfaced, from `low` through `urgent`.
	Priority constants.NotificationPriority `json:"priority" validate:"required"`
	// The resource the announcement is about, which the client can link to.
	Resource *Entity `json:"resource" expandable:"true"`
	// When the announcement becomes visible in the feed.
	//
	// An announcement scheduled for the future is not returned by the announcement endpoints until this time passes.
	PublishAt time.Time `json:"publish_at" validate:"required"`
	// When the announcement stops being shown.
	//
	// Once it expires the announcement leaves every user's feed and can no longer be retrieved; an announcement with no expiry stays until each user dismisses it.
	ExpiresAt *time.Time `json:"expires_at"`
	// When the calling user first saw the announcement.
	SeenAt *time.Time `json:"seen_at"`
	// When the calling user opened the announcement.
	ReadAt *time.Time `json:"read_at"`
	// When the calling user dismissed the announcement.
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
