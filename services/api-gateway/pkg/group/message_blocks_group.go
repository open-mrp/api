package httpgroup

import (
	"fmt"

	blockep "github.com/open-mrp/api/services/api-gateway/endpoints/message-blocks"
	grpcclient "github.com/open-mrp/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
)

type MessageBlocksEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type MessageBlocksEndpointGroupConfig struct {
	// NotificationClient (required) is the notification-service gRPC client.
	NotificationClient *grpcclient.NotificationServiceClient
}

func (c *MessageBlocksEndpointGroupConfig) validate() error {
	if c.NotificationClient == nil {
		return fmt.Errorf("message blocks endpoint group: notification client is required")
	}
	return nil
}

func (*MessageBlocksEndpointGroup) Materialize(config *MessageBlocksEndpointGroupConfig) *MessageBlocksEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	blockSvc := blockep.NewBlockSvc(&blockep.BlockSvcConfig{
		ChatClient: config.NotificationClient.ChatClient,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Message Blocks",
		Description:  "Block and unblock users from direct messaging.",
		ResourceType: &apiresource.MessagingBlock{},
	}

	inner.Endpoints = []apiendpoint.APIEndpointer{
		apiendpoint.From(&blockep.BlockEndpoint{}).WithService(inner, blockSvc),
		apiendpoint.From(&blockep.UnblockEndpoint{}).WithService(inner, blockSvc),
		apiendpoint.From(&blockep.ListBlocksEndpoint{}).WithService(inner, blockSvc),
	}

	return &MessageBlocksEndpointGroup{inner}
}
