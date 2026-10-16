package httpgroup

import (
	"fmt"

	notificationdefaultsep "github.com/open-mrp/api/services/api-gateway/endpoints/notification-defaults"
	subscriptionep "github.com/open-mrp/api/services/api-gateway/endpoints/subscriptions"
	grpcclient "github.com/open-mrp/api/services/api-gateway/grpc-client"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

type SubscriptionsEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type SubscriptionsEndpointGroupConfig struct {
	// NotificationClient (required) is the notification-service gRPC client.
	NotificationClient *grpcclient.NotificationServiceClient
}

func (c *SubscriptionsEndpointGroupConfig) validate() error {
	if c.NotificationClient == nil {
		return fmt.Errorf("subscriptions endpoint group: notification client is required")
	}
	return nil
}

func (*SubscriptionsEndpointGroup) Materialize(config *SubscriptionsEndpointGroupConfig) *SubscriptionsEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}
	svc := subscriptionep.NewSvc(&subscriptionep.SvcConfig{MessagingClient: config.NotificationClient.MessagingClient})
	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Subscriptions",
		Description:  "Which records you're notified about, and following or unfollowing a record.",
		ResourceType: &apiresource.Subscription{},
	}
	inner.Endpoints = []apiendpoint.APIEndpointer{
		apiendpoint.From(&subscriptionep.ListSubscriptionsEndpoint{}).WithService(inner, svc),
		apiendpoint.From(&subscriptionep.CreateSubscriptionEndpoint{}).WithService(inner, svc),
		apiendpoint.From(&subscriptionep.RetrieveSubscriptionEndpoint{}).WithService(inner, svc),
		apiendpoint.From(&subscriptionep.SubscribeEndpoint{}).WithService(inner, svc),
		apiendpoint.From(&subscriptionep.UnsubscribeEndpoint{}).WithService(inner, svc),
	}
	return &SubscriptionsEndpointGroup{inner}
}

type NotificationDefaultsEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type NotificationDefaultsEndpointGroupConfig struct {
	// NotificationClient (required) is the notification-service gRPC client.
	NotificationClient *grpcclient.NotificationServiceClient
}

func (c *NotificationDefaultsEndpointGroupConfig) validate() error {
	if c.NotificationClient == nil {
		return fmt.Errorf("notification defaults endpoint group: notification client is required")
	}
	return nil
}

func (*NotificationDefaultsEndpointGroup) Materialize(config *NotificationDefaultsEndpointGroupConfig) *NotificationDefaultsEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}
	svc := notificationdefaultsep.NewSvc(&notificationdefaultsep.SvcConfig{MessagingClient: config.NotificationClient.MessagingClient})
	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Notification Defaults",
		Description:  "Your team's starting notification settings.",
		ResourceType: &apiresource.NotificationDefaults{},
	}
	inner.Endpoints = []apiendpoint.APIEndpointer{
		apiendpoint.From(&notificationdefaultsep.RetrieveNotificationDefaultsEndpoint{}).WithService(inner, svc),
		apiendpoint.From(&notificationdefaultsep.UpdateNotificationDefaultsEndpoint{}).WithService(inner, svc),
	}
	return &NotificationDefaultsEndpointGroup{inner}
}
