package httpgroup

import (
	"fmt"

	conversationep "github.com/augno/api/services/api-gateway/endpoints/conversations"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

type ConversationsEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type ConversationsEndpointGroupConfig struct {
	// NotificationClient (required) is the notification-service gRPC client.
	NotificationClient *grpcclient.NotificationServiceClient
}

func (c *ConversationsEndpointGroupConfig) validate() error {
	if c.NotificationClient == nil {
		return fmt.Errorf("conversations endpoint group: notification client is required")
	}
	return nil
}

func (*ConversationsEndpointGroup) Materialize(config *ConversationsEndpointGroupConfig) *ConversationsEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	conversationSvc := conversationep.NewConversationSvc(&conversationep.ConversationSvcConfig{
		ChatClient: config.NotificationClient.ChatClient,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Conversations",
		Description:  "Create conversations, send and read messages (1:1 direct messages).",
		ResourceType: &apiresource.Conversation{},
	}

	inner.Endpoints = []apiendpoint.APIEndpointer{
		apiendpoint.From(&conversationep.CreateConversationEndpoint{}).WithService(inner, conversationSvc),
		apiendpoint.From(&conversationep.ListConversationsEndpoint{}).WithService(inner, conversationSvc),
		apiendpoint.From(&conversationep.RetrieveConversationEndpoint{}).WithService(inner, conversationSvc),
		apiendpoint.From(&conversationep.ContactSupportEndpoint{}).WithService(inner, conversationSvc),
		apiendpoint.From(&conversationep.SupportAvailabilityEndpoint{}).WithService(inner, conversationSvc),
		apiendpoint.From(&conversationep.SetLegalHoldEndpoint{}).WithService(inner, conversationSvc),
		apiendpoint.From(&conversationep.RedactConversationEndpoint{}).WithService(inner, conversationSvc),
		apiendpoint.From(&conversationep.MarkConversationReadEndpoint{}).WithService(inner, conversationSvc),
		apiendpoint.From(&conversationep.UpdateConversationEndpoint{}).WithService(inner, conversationSvc),
		apiendpoint.From(&conversationep.ArchiveConversationEndpoint{}).WithService(inner, conversationSvc),
		apiendpoint.From(&conversationep.UnarchiveConversationEndpoint{}).WithService(inner, conversationSvc),
		apiendpoint.From(&conversationep.LeaveConversationEndpoint{}).WithService(inner, conversationSvc),
		apiendpoint.From(&conversationep.HideConversationEndpoint{}).WithService(inner, conversationSvc),
		apiendpoint.From(&conversationep.UnhideConversationEndpoint{}).WithService(inner, conversationSvc),
		apiendpoint.From(&conversationep.MuteConversationEndpoint{}).WithService(inner, conversationSvc),
		apiendpoint.From(&conversationep.UnmuteConversationEndpoint{}).WithService(inner, conversationSvc),
		apiendpoint.From(&conversationep.TypingEndpoint{}).WithService(inner, conversationSvc),
		apiendpoint.From(&conversationep.SetWorkflowStatusEndpoint{}).WithService(inner, conversationSvc),
		apiendpoint.From(&conversationep.AssignConversationEndpoint{}).WithService(inner, conversationSvc),
		apiendpoint.From(&conversationep.ReportConversationEndpoint{}).WithService(inner, conversationSvc),
		apiendpoint.From(&conversationep.AddConversationLinkEndpoint{}).WithService(inner, conversationSvc),
		apiendpoint.From(&conversationep.RemoveConversationLinkEndpoint{}).WithService(inner, conversationSvc),
		apiendpoint.From(&conversationep.ListConversationLinksEndpoint{}).WithService(inner, conversationSvc),
	}

	return &ConversationsEndpointGroup{inner}
}
