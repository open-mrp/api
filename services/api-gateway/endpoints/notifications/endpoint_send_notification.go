package notificationep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// Request to send an in-app notification.
//
// The target determines whether it is delivered to a single user or broadcast to every user in an account. This endpoint is internal/admin only.
type SendNotificationRequest struct {
	// Category of the notification.
	Category constants.NotificationCategory `json:"category" validate:"required"`
	// Who to send to: an `account_user` for a per-user notification, or an `account` to broadcast an announcement to every user in it.
	Target apiresource.NotificationTargetInput `json:"target" validate:"required"`
	// Human-readable title.
	Title string `json:"title" validate:"required"`
	// Preview/body text.
	Body field.Optional[string] `json:"body,omitzero"`
	// Delivery priority.
	Priority field.Optional[constants.NotificationPriority] `json:"priority,omitzero" default:"normal"`
	// Object type of the resource this notification should link to.
	//
	// Set together with `link_resource_id` to point the notification at something the recipient can open.
	LinkResourceType field.Optional[constants.ObjectType] `json:"link_resource_type,omitzero"`
	// ID of the resource this notification should link to.
	LinkResourceID field.Optional[string] `json:"link_resource_id,omitzero"`
}

var sampleSendNotificationBody = "Order #1042 was updated."

var sampleSendNotificationRequest = &SendNotificationRequest{
	Category: constants.NotificationCategoryOrderUpdated,
	Target: apiresource.NotificationTargetInput{
		Type: constants.NotificationTargetTypeAccountUser,
		ID:   apiresource.SampleAccountUserID,
	},
	Title:            "Order updated",
	Body:             field.Some(sampleSendNotificationBody),
	Priority:         field.Some(constants.NotificationPriorityHigh),
	LinkResourceType: field.Some(constants.ObjectTypeSalesOrder),
	LinkResourceID:   field.Some(apiresource.SampleSalesOrderID),
}

func (*SendNotificationRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleSendNotificationRequest)
}

// Sends an in-app notification to a single user or broadcasts it to every user in an account.
//
// Delivery is asynchronous: the notification is fanned out to its recipients and pushed to connected clients in real time.
type SendNotificationEndpoint struct{}

func (e *SendNotificationEndpoint) Materialize() *apiendpoint.APIEndpoint[*SendNotificationRequest, *apiresource.NotificationSendResult] {
	return (&apiendpoint.APIEndpoint[*SendNotificationRequest, *apiresource.NotificationSendResult]{
		Title:               "Send Notification",
		Method:              http.MethodPost,
		ContentType:         "application/json",
		Route:               "/v1/messaging/notifications",
		SuccessStatusCode:   http.StatusAccepted,
		Public:              true,
		AgentTool:           true,
		Preview:             true,
		ObjectType:          constants.ObjectTypeNotificationSendResult,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainAlerts, Action: types.ActionCreate}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *SendNotificationRequest) (*apiresource.NotificationSendResult, *apierror.APIError) {
			return svc.(NotificationSvc).SendNotification
		},
	})
}
