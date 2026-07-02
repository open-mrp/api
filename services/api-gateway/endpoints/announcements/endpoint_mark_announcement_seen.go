package announcementep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to transition the caller's receipt for a single announcement.
type MarkAnnouncementRequest struct {
	// Announcement ID.
	AnnouncementID string `path:"id" validate:"required"`
}

// Marks an announcement as seen for the calling actor.
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
