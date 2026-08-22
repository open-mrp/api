package httpgroup

import (
	"fmt"

	participantep "github.com/open-mrp/api/services/api-gateway/endpoints/conversation-participants"
	grpcclient "github.com/open-mrp/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
)

type ConversationParticipantsEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type ConversationParticipantsEndpointGroupConfig struct {
	// NotificationClient (required) is the notification-service gRPC client.
	NotificationClient *grpcclient.NotificationServiceClient
}

func (c *ConversationParticipantsEndpointGroupConfig) validate() error {
	if c.NotificationClient == nil {
		return fmt.Errorf("conversation participants endpoint group: notification client is required")
	}
	return nil
}

func (*ConversationParticipantsEndpointGroup) Materialize(config *ConversationParticipantsEndpointGroupConfig) *ConversationParticipantsEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	participantSvc := participantep.NewParticipantSvc(&participantep.ParticipantSvcConfig{
		ChatClient: config.NotificationClient.ChatClient,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Conversation Participants",
		Description:  "Add, remove, and manage participants (including agents) in a conversation.",
		ResourceType: &apiresource.ConversationParticipant{},
	}

	inner.Endpoints = []apiendpoint.APIEndpointer{
		apiendpoint.From(&participantep.AddParticipantEndpoint{}).WithService(inner, participantSvc),
		apiendpoint.From(&participantep.RemoveParticipantEndpoint{}).WithService(inner, participantSvc),
		apiendpoint.From(&participantep.UpdateParticipantRoleEndpoint{}).WithService(inner, participantSvc),
		apiendpoint.From(&participantep.AddAgentParticipantEndpoint{}).WithService(inner, participantSvc),
		apiendpoint.From(&participantep.RemoveAgentParticipantEndpoint{}).WithService(inner, participantSvc),
	}

	return &ConversationParticipantsEndpointGroup{inner}
}
