package httpgroup

import (
	"fmt"

	supportrouteep "github.com/augno/api/services/api-gateway/endpoints/support-routes"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

type SupportRoutesEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type SupportRoutesEndpointGroupConfig struct {
	// NotificationClient (required) is the notification-service gRPC client.
	NotificationClient *grpcclient.NotificationServiceClient
}

func (c *SupportRoutesEndpointGroupConfig) validate() error {
	if c.NotificationClient == nil {
		return fmt.Errorf("support routes endpoint group: notification client is required")
	}
	return nil
}

func (*SupportRoutesEndpointGroup) Materialize(config *SupportRoutesEndpointGroupConfig) *SupportRoutesEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	supportRouteSvc := supportrouteep.NewSupportRouteSvc(&supportrouteep.SupportRouteSvcConfig{
		ChatClient: config.NotificationClient.ChatClient,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Support Routes",
		Description:  "Designate the group conversation that handles a relationship's inbound support, so customer support messages reach a deterministic set of recipients.",
		ResourceType: &apiresource.SupportRoute{},
	}

	inner.Endpoints = []apiendpoint.APIEndpointer{
		apiendpoint.From(&supportrouteep.SetSupportRouteEndpoint{}).WithService(inner, supportRouteSvc),
		apiendpoint.From(&supportrouteep.ClearSupportRouteEndpoint{}).WithService(inner, supportRouteSvc),
		apiendpoint.From(&supportrouteep.GetSupportRouteEndpoint{}).WithService(inner, supportRouteSvc),
	}

	return &SupportRoutesEndpointGroup{inner}
}
