package httpgroup

import (
	"fmt"

	announcementep "github.com/augno/api/services/api-gateway/endpoints/announcements"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

type AnnouncementsEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type AnnouncementsEndpointGroupConfig struct {
	// NotificationClient (required) is the notification-service gRPC client.
	NotificationClient *grpcclient.NotificationServiceClient
}

func (c *AnnouncementsEndpointGroupConfig) validate() error {
	if c.NotificationClient == nil {
		return fmt.Errorf("announcements endpoint group: notification client is required")
	}
	return nil
}

func (*AnnouncementsEndpointGroup) Materialize(config *AnnouncementsEndpointGroupConfig) *AnnouncementsEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	announcementSvc := announcementep.NewAnnouncementSvc(&announcementep.AnnouncementSvcConfig{
		MessagingClient: config.NotificationClient.MessagingClient,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Announcements",
		Description:  "List, read, and manage broadcast announcements.",
		ResourceType: &apiresource.Announcement{},
	}

	inner.Endpoints = []apiendpoint.APIEndpointer{
		apiendpoint.From(&announcementep.ListAnnouncementsEndpoint{}).WithService(inner, announcementSvc),
		apiendpoint.From(&announcementep.RetrieveAnnouncementEndpoint{}).WithService(inner, announcementSvc),
		apiendpoint.From(&announcementep.MarkAnnouncementSeenEndpoint{}).WithService(inner, announcementSvc),
		apiendpoint.From(&announcementep.MarkAnnouncementReadEndpoint{}).WithService(inner, announcementSvc),
		apiendpoint.From(&announcementep.MarkAnnouncementDismissedEndpoint{}).WithService(inner, announcementSvc),
	}

	return &AnnouncementsEndpointGroup{inner}
}
