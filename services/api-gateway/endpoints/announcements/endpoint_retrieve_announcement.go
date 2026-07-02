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

// Request to retrieve a single active announcement visible to the caller.
type RetrieveAnnouncementRequest struct {
	// Announcement ID.
	AnnouncementID string `path:"id" validate:"required"`
}

// Returns one active announcement by ID.
type RetrieveAnnouncementEndpoint struct{}

func (e *RetrieveAnnouncementEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveAnnouncementRequest, *apiresource.Announcement] {
	return (&apiendpoint.APIEndpoint[*RetrieveAnnouncementRequest, *apiresource.Announcement]{
		Title:               "Retrieve Announcement",
		Method:              http.MethodGet,
		ContentType:         "application/json",
		Route:               "/v1/messaging/announcements/{id}",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		Preview:             true,
		ObjectType:          constants.ObjectTypeAnnouncement,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionRead}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveAnnouncementRequest) (*apiresource.Announcement, *apierror.APIError) {
			return svc.(AnnouncementSvc).GetAnnouncement
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAnnouncement,
			Fields:     []string{"resource"},
		}),
	})
}
