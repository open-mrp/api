package httpgroup

import (
	"fmt"

	messaginggroupep "github.com/augno/api/services/api-gateway/endpoints/messaging-groups"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

type MessagingGroupsEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type MessagingGroupsEndpointGroupConfig struct {
	// NotificationClient (required) is the notification-service gRPC client.
	NotificationClient *grpcclient.NotificationServiceClient
}

func (c *MessagingGroupsEndpointGroupConfig) validate() error {
	if c.NotificationClient == nil {
		return fmt.Errorf("messaging groups endpoint group: notification client is required")
	}
	return nil
}

func (*MessagingGroupsEndpointGroup) Materialize(config *MessagingGroupsEndpointGroupConfig) *MessagingGroupsEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	groupSvc := messaginggroupep.NewMessagingGroupSvc(&messaginggroupep.MessagingGroupSvcConfig{
		ChatClient: config.NotificationClient.ChatClient,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Messaging Groups",
		Description:  "Create and manage reusable rosters (named member sets) that seed many conversations.",
		ResourceType: &apiresource.MessagingGroup{},
	}

	inner.Endpoints = []apiendpoint.APIEndpointer{
		apiendpoint.From(&messaginggroupep.CreateMessagingGroupEndpoint{}).WithService(inner, groupSvc),
		apiendpoint.From(&messaginggroupep.ListMessagingGroupsEndpoint{}).WithService(inner, groupSvc),
		apiendpoint.From(&messaginggroupep.GetMessagingGroupEndpoint{}).WithService(inner, groupSvc),
		apiendpoint.From(&messaginggroupep.UpdateMessagingGroupEndpoint{}).WithService(inner, groupSvc),
		apiendpoint.From(&messaginggroupep.DeleteMessagingGroupEndpoint{}).WithService(inner, groupSvc),
		apiendpoint.From(&messaginggroupep.AddMessagingGroupMemberEndpoint{}).WithService(inner, groupSvc),
		apiendpoint.From(&messaginggroupep.RemoveMessagingGroupMemberEndpoint{}).WithService(inner, groupSvc),
	}

	return &MessagingGroupsEndpointGroup{inner}
}
