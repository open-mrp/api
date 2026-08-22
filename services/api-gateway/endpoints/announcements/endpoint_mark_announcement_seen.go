package announcementep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to advance the calling user's state for a single announcement.
type MarkAnnouncementRequest struct {
	// Announcement ID.
	AnnouncementID string `path:"id" validate:"required"`
}

// Marks an announcement as seen for the calling user, as when it is surfaced to them without being opened.
//
// Seeing an announcement clears it from the caller's unread bell total but leaves it in the feed, and only affects the caller: everyone else in the account keeps their own state. Repeating the call keeps the original seen time. A caller with no user of their own in the account, such as an API key, has no state to record and gets a not-found error.
type MarkAnnouncementSeenEndpoint struct{}

func (e *MarkAnnouncementSeenEndpoint) Materialize() *apiendpoint.APIEndpoint[*MarkAnnouncementRequest, *apiresource.Announcement] {
	return (&apiendpoint.APIEndpoint[*MarkAnnouncementRequest, *apiresource.Announcement]{
		Title:               "Mark Announcement Seen",
		Method:              http.MethodPost,
		ContentType:         "application/json",
		Route:               "/v1/messaging/announcements/{id}/actions/seen",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		Preview:             true,
		ObjectType:          constants.ObjectTypeAnnouncement,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionUpdate}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *MarkAnnouncementRequest) (*apiresource.Announcement, *apierror.APIError) {
			return svc.(AnnouncementSvc).MarkAnnouncementSeen
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAnnouncement,
			Fields:     []string{"resource"},
		}),
	})
}
