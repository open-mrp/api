package httpgroup

import (
	"fmt"

	attachmentep "github.com/augno/api/services/api-gateway/endpoints/message-attachments"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

type MessageAttachmentsEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type MessageAttachmentsEndpointGroupConfig struct {
	// NotificationClient (required) is the notification-service gRPC client.
	NotificationClient *grpcclient.NotificationServiceClient
}

func (c *MessageAttachmentsEndpointGroupConfig) validate() error {
	if c.NotificationClient == nil {
		return fmt.Errorf("message attachments endpoint group: notification client is required")
	}
	return nil
}

func (*MessageAttachmentsEndpointGroup) Materialize(config *MessageAttachmentsEndpointGroupConfig) *MessageAttachmentsEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	attachmentSvc := attachmentep.NewAttachmentSvc(&attachmentep.AttachmentSvcConfig{
		ChatClient: config.NotificationClient.ChatClient,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Message Attachments",
		Description:  "Create presigned upload targets for message attachments.",
		ResourceType: &apiresource.AttachmentUploadTarget{},
	}

	inner.Endpoints = []apiendpoint.APIEndpointer{
		apiendpoint.From(&attachmentep.CreateAttachmentUploadURLEndpoint{}).WithService(inner, attachmentSvc),
	}

	return &MessageAttachmentsEndpointGroup{inner}
}
