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

// Marks an announcement as read for the calling user, as when they open it.
//
// Reading also marks the announcement seen if it was not already, and leaves it in the feed until it is dismissed. Repeating the call keeps the original read time. A caller with no user of their own in the account, such as an API key, has no state to record and gets a not-found error.
type MarkAnnouncementReadEndpoint struct{}

func (e *MarkAnnouncementReadEndpoint) Materialize() *apiendpoint.APIEndpoint[*MarkAnnouncementRequest, *apiresource.Announcement] {
	return (&apiendpoint.APIEndpoint[*MarkAnnouncementRequest, *apiresource.Announcement]{
		Title:               "Mark Announcement Read",
		Method:              http.MethodPost,
		ContentType:         "application/json",
		Route:               "/v1/messaging/announcements/{id}/actions/read",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		Preview:             true,
		ObjectType:          constants.ObjectTypeAnnouncement,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionUpdate}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *MarkAnnouncementRequest) (*apiresource.Announcement, *apierror.APIError) {
			return svc.(AnnouncementSvc).MarkAnnouncementRead
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAnnouncement,
			Fields:     []string{"resource"},
		}),
	})
}
