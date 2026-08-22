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

// Dismisses an announcement for the calling user, removing it from their feed.
//
// The announcement itself is not deleted: it stays retrievable by ID and remains in every other user's feed until they dismiss it too. Dismissing an already-dismissed announcement keeps the original dismissal time. A caller with no user of their own in the account, such as an API key, has no state to record and gets a not-found error.
type MarkAnnouncementDismissedEndpoint struct{}

func (e *MarkAnnouncementDismissedEndpoint) Materialize() *apiendpoint.APIEndpoint[*MarkAnnouncementRequest, *apiresource.Announcement] {
	return (&apiendpoint.APIEndpoint[*MarkAnnouncementRequest, *apiresource.Announcement]{
		Title:               "Mark Announcement Dismissed",
		Method:              http.MethodPost,
		ContentType:         "application/json",
		Route:               "/v1/messaging/announcements/{id}/actions/dismiss",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		Preview:             true,
		ObjectType:          constants.ObjectTypeAnnouncement,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionUpdate}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *MarkAnnouncementRequest) (*apiresource.Announcement, *apierror.APIError) {
			return svc.(AnnouncementSvc).MarkAnnouncementDismissed
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAnnouncement,
			Fields:     []string{"resource"},
		}),
	})
}
