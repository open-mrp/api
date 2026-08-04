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
// The target decides whether the notification goes to one member of the account or to everyone in it.
type SendNotificationRequest struct {
	// The kind of event the notification represents, such as `order.updated`.
	//
	// Categories are how clients group and filter the feed, so reuse an existing one where it fits.
	Category constants.NotificationCategory `json:"category" validate:"required"`
	// Who to send to: an `account_user` for a personal notification, or an `account` to announce to everyone in it.
	Target apiresource.NotificationTargetInput `json:"target" validate:"required"`
	// Short headline shown in the recipient's feed.
	Title string `json:"title" validate:"required"`
	// Supporting detail shown beneath the title.
	Body field.Optional[string] `json:"body,omitzero"`
	// How prominently the notification should be surfaced, from `low` through `urgent`.
	Priority field.Optional[constants.NotificationPriority] `json:"priority,omitzero" default:"normal"`
	// Type of the resource the notification should link to, such as `sales_order`.
	//
	// Set it together with `link_resource_id` to point the notification at something the recipient can open; supplying only one of the two produces a notification with no link.
	LinkResourceType field.Optional[constants.ObjectType] `json:"link_resource_type,omitzero"`
	// ID of the resource the notification should link to.
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

// Sends an in-app notification to a single member of an account, or announces it to everyone in the account.
//
// A send to one member is attributed to the authenticated caller, so the recipient sees who sent it. It is accepted and then fanned out, so it reaches the recipient's feed and their connected clients shortly after the response.
//
// An announcement to the whole account is stored as the request is accepted, carries no sender, and may only target the account you are currently acting in.
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
