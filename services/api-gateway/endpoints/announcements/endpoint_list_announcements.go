package announcementep

import (
	"context"
	"net/http"

	"github.com/open-mrp/apikit/object"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

// Request to list the caller's active announcements.
type ListAnnouncementsRequest struct {
	object.ListRequest
	// Free-text search term used to filter results.
	//
	// Which fields are matched against the term varies by endpoint.
	Query *string `query:"q" validate:"omitempty,max=500"`
}

// Lists the announcements currently active for the caller, newest first.
//
// The feed covers announcements broadcast to the account being acted in together with platform-wide announcements from OpenMRP. Announcements the caller has dismissed are left out, as are any that are scheduled for later or have already expired.
type ListAnnouncementsEndpoint struct{}

func (e *ListAnnouncementsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListAnnouncementsRequest, *apiresource.List[apiresource.Announcement]] {
	return (&apiendpoint.APIEndpoint[*ListAnnouncementsRequest, *apiresource.List[apiresource.Announcement]]{
		Title:             "List Announcements",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/messaging/announcements",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Preview:           true,
		ObjectType:        constants.ObjectTypeAnnouncement,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionRead}}},
		Extras:            apiendpoint.APIEndpointExtras{HideFromRequestLog: true},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListAnnouncementsRequest) (*apiresource.List[apiresource.Announcement], *apierror.APIError) {
			return svc.(AnnouncementSvc).ListAnnouncements
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAnnouncement,
			Fields:     []string{"resource"},
		}),
	})
}
