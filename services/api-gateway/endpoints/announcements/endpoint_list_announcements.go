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

// Request to list the caller's active announcements.
type ListAnnouncementsRequest struct {
	apiresource.PaginationRequest
}

// Lists the announcements currently active for the caller, newest first.
//
// The feed covers announcements broadcast to the account being acted in together with platform-wide announcements from Augno. Announcements the caller has dismissed are left out, as are any that are scheduled for later or have already expired.
type ListAnnouncementsEndpoint struct{}

func (e *ListAnnouncementsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListAnnouncementsRequest, *apiresource.List[apiresource.Announcement]] {
	return (&apiendpoint.APIEndpoint[*ListAnnouncementsRequest, *apiresource.List[apiresource.Announcement]]{
		Title:               "List Announcements",
		Method:              http.MethodGet,
		ContentType:         "application/json",
		Route:               "/v1/messaging/announcements",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		Preview:             true,
		ObjectType:          constants.ObjectTypeAnnouncement,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionRead}},
		Extras:              apiendpoint.APIEndpointExtras{HideFromRequestLog: true},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListAnnouncementsRequest) (*apiresource.List[apiresource.Announcement], *apierror.APIError) {
			return svc.(AnnouncementSvc).ListAnnouncements
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAnnouncement,
			Fields:     []string{"resource"},
		}),
	})
}
