package httpgroup

import (
	"fmt"

	notificationpreferenceep "github.com/augno/api/services/api-gateway/endpoints/notification-preferences"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
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
		Description:  "Manage per-category notification channel preferences (in-app, email, push).",
		ResourceType: &apiresource.NotificationPreference{},
	}

	inner.Endpoints = []apiendpoint.APIEndpointer{
		apiendpoint.From(&notificationpreferenceep.ListNotificationPreferencesEndpoint{}).WithService(inner, preferenceSvc),
		apiendpoint.From(&notificationpreferenceep.UpsertNotificationPreferenceEndpoint{}).WithService(inner, preferenceSvc),
	}

	return &NotificationPreferencesEndpointGroup{inner}
}
