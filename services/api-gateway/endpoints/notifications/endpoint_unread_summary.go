package notificationep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request for the caller's cross-account unread summary.
type UnreadSummaryRequest struct{}

// Returns the caller's unread totals broken down by account, covering every account they belong to and not just the one they are acting in.
//
// Use it to show a user that activity is waiting for them elsewhere before they switch accounts. Each tally counts unseen notifications and unseen account announcements together.
type UnreadSummaryEndpoint struct{}

func (e *UnreadSummaryEndpoint) Materialize() *apiendpoint.APIEndpoint[*UnreadSummaryRequest, *apiresource.NotificationUnreadSummary] {
	return (&apiendpoint.APIEndpoint[*UnreadSummaryRequest, *apiresource.NotificationUnreadSummary]{
		Title:               "Get Cross-Account Unread Summary",
		Method:              http.MethodGet,
		ContentType:         "application/json",
		Route:               "/v1/messaging/notifications/unread-summary",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		Preview:             true,
		ObjectType:          constants.ObjectTypeNotificationUnreadSummary,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionRead}},
		Extras:              apiendpoint.APIEndpointExtras{HideFromRequestLog: true},
		ServiceHandler: func(svc any) func(ctx context.Context, req *UnreadSummaryRequest) (*apiresource.NotificationUnreadSummary, *apierror.APIError) {
			return svc.(NotificationSvc).GetUnreadSummary
		},
	})
}
