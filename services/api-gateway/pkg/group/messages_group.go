package httpgroup

import (
	"fmt"

	messageep "github.com/open-mrp/api/services/api-gateway/endpoints/messages"
	grpcclient "github.com/open-mrp/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
)

type MessagesEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type MessagesEndpointGroupConfig struct {
	// NotificationClient (required) is the notification-service gRPC client.
	NotificationClient *grpcclient.NotificationServiceClient
}

func (c *MessagesEndpointGroupConfig) validate() error {
	if c.NotificationClient == nil {
		return fmt.Errorf("messages endpoint group: notification client is required")
	}
	return nil
}

func (*MessagesEndpointGroup) Materialize(config *MessagesEndpointGroupConfig) *MessagesEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	messageSvc := messageep.NewMessageSvc(&messageep.MessageSvcConfig{
		ChatClient: config.NotificationClient.ChatClient,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Messages",
		Description:  "Send, list, edit, and delete chat messages.",
		ResourceType: &apiresource.Message{},
	}

	inner.Endpoints = []apiendpoint.APIEndpointer{
		apiendpoint.From(&messageep.SendMessageEndpoint{}).WithService(inner, messageSvc),
		apiendpoint.From(&messageep.ListMessagesEndpoint{}).WithService(inner, messageSvc),
		apiendpoint.From(&messageep.UpdateDraftEndpoint{}).WithService(inner, messageSvc),
		apiendpoint.From(&messageep.ApproveSendDraftEndpoint{}).WithService(inner, messageSvc),
		apiendpoint.From(&messageep.RejectDraftEndpoint{}).WithService(inner, messageSvc),
		apiendpoint.From(&messageep.CancelScheduledEndpoint{}).WithService(inner, messageSvc),
	}

	return &MessagesEndpointGroup{inner}
}
