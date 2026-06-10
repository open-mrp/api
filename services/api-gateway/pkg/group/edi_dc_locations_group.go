package httpgroup

import (
	"fmt"

	edidclocationep "github.com/augno/api/services/api-gateway/endpoints/edi-dc-locations"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

type EDIDCLocationsEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type EDIDCLocationsEndpointGroupConfig struct {
	// CoreClient (required) is the core-service gRPC client.
	CoreClient *grpcclient.CoreServiceClient
}

func (c *EDIDCLocationsEndpointGroupConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("edi dc locations endpoint group: core client is required")
	}
	return nil
}

func (*EDIDCLocationsEndpointGroup) Materialize(config *EDIDCLocationsEndpointGroupConfig) *EDIDCLocationsEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	svc := edidclocationep.NewEDIDCLocationSvc(&edidclocationep.EDIDCLocationSvcConfig{
		CoreClient: config.CoreClient.Client,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "DC Locations Management",
		Description:  "List and manage DC locations.",
		ResourceType: &apiresource.DCLocation{},
	}

	listEndpoint := apiendpoint.From(&edidclocationep.ListDCLocationsEndpoint{}).WithService(inner, svc)
	retrieveEndpoint := apiendpoint.From(&edidclocationep.RetrieveDCLocationEndpoint{}).WithService(inner, svc)
	createEndpoint := apiendpoint.From(&edidclocationep.CreateDCLocationEndpoint{}).WithService(inner, svc)
	updateEndpoint := apiendpoint.From(&edidclocationep.UpdateDCLocationEndpoint{}).WithService(inner, svc)
	deleteEndpoint := apiendpoint.From(&edidclocationep.DeleteDCLocationEndpoint{}).WithService(inner, svc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		listEndpoint,
		retrieveEndpoint,
		createEndpoint,
		updateEndpoint,
		deleteEndpoint,
	}

	return &EDIDCLocationsEndpointGroup{inner}
}
