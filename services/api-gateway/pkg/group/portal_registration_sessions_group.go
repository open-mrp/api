package httpgroup

import (
	"fmt"

	portalregsessionep "github.com/augno/api/services/api-gateway/endpoints/portal-registration-sessions"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

type PortalRegistrationSessionsEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type PortalRegistrationSessionsEndpointGroupConfig struct {
	// CoreClient (required) is the core-service gRPC client.
	CoreClient *grpcclient.CoreServiceClient
}

func (c *PortalRegistrationSessionsEndpointGroupConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("portal registration sessions endpoint group: core client is required")
	}
	return nil
}

func (*PortalRegistrationSessionsEndpointGroup) Materialize(config *PortalRegistrationSessionsEndpointGroupConfig) *PortalRegistrationSessionsEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	svc := portalregsessionep.NewPortalRegistrationSessionSvc(&portalregsessionep.PortalRegistrationSessionSvcConfig{
		CoreClient: config.CoreClient.Client,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Portal Registration Sessions",
		Description:  "Session-based registration of a buyer into a seller's customer portal: start or resume, advance step by step, then complete or abandon.",
		ResourceType: &apiresource.PortalRegistrationSession{},
	}

	inner.Endpoints = []apiendpoint.APIEndpointer{
		apiendpoint.From(&portalregsessionep.CreateOrResumePortalRegistrationSessionEndpoint{}).WithService(inner, svc),
		apiendpoint.From(&portalregsessionep.GetPortalRegistrationSessionEndpoint{}).WithService(inner, svc),
		apiendpoint.From(&portalregsessionep.UpdatePortalRegistrationSessionEndpoint{}).WithService(inner, svc),
		apiendpoint.From(&portalregsessionep.CompletePortalRegistrationSessionEndpoint{}).WithService(inner, svc),
		apiendpoint.From(&portalregsessionep.AbandonPortalRegistrationSessionEndpoint{}).WithService(inner, svc),
	}

	return &PortalRegistrationSessionsEndpointGroup{inner}
}
