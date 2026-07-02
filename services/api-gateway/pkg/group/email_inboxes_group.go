package httpgroup

import (
	"fmt"

	emailbridgeep "github.com/augno/api/services/api-gateway/endpoints/email-bridge"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

type EmailInboxesEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type EmailInboxesEndpointGroupConfig struct {
	// NotificationClient (required) is the notification-service gRPC client.
	NotificationClient *grpcclient.NotificationServiceClient
}

func (c *EmailInboxesEndpointGroupConfig) validate() error {
	if c.NotificationClient == nil {
		return fmt.Errorf("email inboxes endpoint group: notification client is required")
	}
	return nil
}

func (*EmailInboxesEndpointGroup) Materialize(config *EmailInboxesEndpointGroupConfig) *EmailInboxesEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	emailBridgeSvc := emailbridgeep.NewEmailBridgeSvc(&emailbridgeep.EmailBridgeSvcConfig{
		EmailBridgeClient: config.NotificationClient.EmailBridgeClient,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Email Inboxes",
		Description:  "Provision and manage routable email inboxes that bind inbound mail to chat conversations and send agent replies.",
		ResourceType: &apiresource.EmailInbox{},
	}

	inner.Endpoints = []apiendpoint.APIEndpointer{
		apiendpoint.From(&emailbridgeep.CreateEmailInboxEndpoint{}).WithService(inner, emailBridgeSvc),
		apiendpoint.From(&emailbridgeep.ListEmailInboxesEndpoint{}).WithService(inner, emailBridgeSvc),
		apiendpoint.From(&emailbridgeep.GetEmailInboxEndpoint{}).WithService(inner, emailBridgeSvc),
		apiendpoint.From(&emailbridgeep.UpdateEmailInboxEndpoint{}).WithService(inner, emailBridgeSvc),
		apiendpoint.From(&emailbridgeep.DeleteEmailInboxEndpoint{}).WithService(inner, emailBridgeSvc),
	}

	return &EmailInboxesEndpointGroup{inner}
}
