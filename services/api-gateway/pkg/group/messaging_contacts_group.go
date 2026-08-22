package httpgroup

import (
	"fmt"

	contactep "github.com/open-mrp/api/services/api-gateway/endpoints/messaging-contacts"
	grpcclient "github.com/open-mrp/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
)

type MessagingContactsEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type MessagingContactsEndpointGroupConfig struct {
	// NotificationClient (required) is the notification-service gRPC client.
	NotificationClient *grpcclient.NotificationServiceClient
}

func (c *MessagingContactsEndpointGroupConfig) validate() error {
	if c.NotificationClient == nil {
		return fmt.Errorf("messaging contacts endpoint group: notification client is required")
	}
	return nil
}

func (*MessagingContactsEndpointGroup) Materialize(config *MessagingContactsEndpointGroupConfig) *MessagingContactsEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	contactSvc := contactep.NewContactSvc(&contactep.ContactSvcConfig{
		ChatClient: config.NotificationClient.ChatClient,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Messaging Contacts",
		Description:  "List messageable contacts (the messaging directory).",
		ResourceType: &apiresource.Actor{},
	}

	inner.Endpoints = []apiendpoint.APIEndpointer{
		apiendpoint.From(&contactep.ListContactsEndpoint{}).WithService(inner, contactSvc),
	}

	return &MessagingContactsEndpointGroup{inner}
}
