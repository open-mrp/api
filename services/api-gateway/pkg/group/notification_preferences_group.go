package httpgroup

import (
	"fmt"

	notificationpreferenceep "github.com/open-mrp/api/services/api-gateway/endpoints/notification-preferences"
	grpcclient "github.com/open-mrp/api/services/api-gateway/grpc-client"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

type NotificationPreferencesEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type NotificationPreferencesEndpointGroupConfig struct {
	// NotificationClient (required) is the notification-service gRPC client.
	NotificationClient *grpcclient.NotificationServiceClient
}

func (c *NotificationPreferencesEndpointGroupConfig) validate() error {
	if c.NotificationClient == nil {
		return fmt.Errorf("notification preferences endpoint group: notification client is required")
	}
	return nil
}

func (*NotificationPreferencesEndpointGroup) Materialize(config *NotificationPreferencesEndpointGroupConfig) *NotificationPreferencesEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	preferenceSvc := notificationpreferenceep.NewNotificationPreferenceSvc(&notificationpreferenceep.NotificationPreferenceSvcConfig{
		MessagingClient: config.NotificationClient.MessagingClient,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Notification Preferences",
		Description:  "Where your notifications go: a default, settings per notification type, automatic subscription and pausing.",
		ResourceType: &apiresource.NotificationPreferences{},
	}

	inner.Endpoints = []apiendpoint.APIEndpointer{
		apiendpoint.From(&notificationpreferenceep.RetrieveNotificationPreferencesEndpoint{}).WithService(inner, preferenceSvc),
		apiendpoint.From(&notificationpreferenceep.UpdateNotificationPreferencesEndpoint{}).WithService(inner, preferenceSvc),
		apiendpoint.From(&notificationpreferenceep.ListNotificationPreferencesEndpoint{}).WithService(inner, preferenceSvc),
		apiendpoint.From(&notificationpreferenceep.UpsertNotificationPreferenceEndpoint{}).WithService(inner, preferenceSvc),
	}

	return &NotificationPreferencesEndpointGroup{inner}
}
