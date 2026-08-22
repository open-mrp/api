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

// Request to retrieve a single active announcement visible to the caller.
type RetrieveAnnouncementRequest struct {
	// Announcement ID.
	AnnouncementID string `path:"id" validate:"required"`
}

// Retrieves a single announcement by ID, with the calling user's own read state.
//
// Only announcements the caller can see are returned: one published to another account, one that has not reached its publish time, or one that has expired is reported as not found. An announcement the caller has dismissed stays retrievable even though it no longer appears in their feed.
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
