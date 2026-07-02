package httpgroup

import (
	"fmt"

	notificationep "github.com/augno/api/services/api-gateway/endpoints/notifications"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

type NotificationsEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type NotificationsEndpointGroupConfig struct {
	// NotificationClient (required) is the notification-service gRPC client.
	NotificationClient *grpcclient.NotificationServiceClient
}

func (c *NotificationsEndpointGroupConfig) validate() error {
	if c.NotificationClient == nil {
		return fmt.Errorf("notifications endpoint group: notification client is required")
	}
	return nil
}

func (*NotificationsEndpointGroup) Materialize(config *NotificationsEndpointGroupConfig) *NotificationsEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	notificationSvc := notificationep.NewNotificationSvc(&notificationep.NotificationSvcConfig{
		MessagingClient: config.NotificationClient.MessagingClient,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Notifications",
		Description:  "List, read, and manage in-app notifications.",
		ResourceType: &apiresource.Notification{},
	}

	inner.Endpoints = []apiendpoint.APIEndpointer{
		apiendpoint.From(&notificationep.SendNotificationEndpoint{}).WithService(inner, notificationSvc),
		apiendpoint.From(&notificationep.ListNotificationsEndpoint{}).WithService(inner, notificationSvc),
		apiendpoint.From(&notificationep.RetrieveNotificationEndpoint{}).WithService(inner, notificationSvc),
		apiendpoint.From(&notificationep.UnreadCountEndpoint{}).WithService(inner, notificationSvc),
		apiendpoint.From(&notificationep.UnreadSummaryEndpoint{}).WithService(inner, notificationSvc),
		apiendpoint.From(&notificationep.MarkSeenEndpoint{}).WithService(inner, notificationSvc),
		apiendpoint.From(&notificationep.MarkReadEndpoint{}).WithService(inner, notificationSvc),
		apiendpoint.From(&notificationep.MarkDismissedEndpoint{}).WithService(inner, notificationSvc),
		apiendpoint.From(&notificationep.MarkAllSeenEndpoint{}).WithService(inner, notificationSvc),
	}

	return &NotificationsEndpointGroup{inner}
}
