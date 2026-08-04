package notificationep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	apierror "github.com/augno/api/shared/errors"
)

// Request to mark all of the caller's notifications as seen.
type MarkAllSeenRequest struct{}

// Marks every one of the caller's unseen notifications as seen in a single call.
//
// The notifications stay in the feed and are not marked read. Account announcements are unaffected and are cleared individually, so the unread total can remain above zero afterwards.
type MarkAllSeenEndpoint struct{}

func (e *MarkAllSeenEndpoint) Materialize() *apiendpoint.APIEndpoint[*MarkAllSeenRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*MarkAllSeenRequest, *apiresource.EmptyResource]{
		Title:               "Mark All Notifications Seen",
		Method:              http.MethodPost,
		ContentType:         "application/json",
		Route:               "/v1/messaging/notifications/actions/mark-all-seen",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		Preview:             true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionUpdate}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *MarkAllSeenRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(NotificationSvc).MarkAllSeen
		},
	})
}
