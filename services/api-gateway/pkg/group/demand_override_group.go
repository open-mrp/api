package httpgroup

import (
	"fmt"

	demandoverridesep "github.com/open-mrp/api/services/api-gateway/endpoints/demand-overrides"
	grpcclient "github.com/open-mrp/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
)

type DemandOverridesEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type DemandOverridesEndpointGroupConfig struct {
	// CoreClient (required) is the core-service gRPC client.
	CoreClient *grpcclient.CoreServiceClient
}

func (c *DemandOverridesEndpointGroupConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("demand overrides endpoint group: core client is required")
	}
	return nil
}

func (*DemandOverridesEndpointGroup) Materialize(config *DemandOverridesEndpointGroupConfig) *DemandOverridesEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	svc := demandoverridesep.NewDemandOverridesSvc(&demandoverridesep.DemandOverridesSvcConfig{
		CoreClient: config.CoreClient.DemandOverride,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Demand Overrides",
		Description:  "Adjust the demand a production schedule plans against. Overrides are how management accounts for demand that sales history cannot see.",
		ResourceType: &apiresource.DemandOverride{},
	}

	listTypesEndpoint := apiendpoint.From(&demandoverridesep.ListDemandOverrideTypesEndpoint{}).WithService(inner, svc)
	listEndpoint := apiendpoint.From(&demandoverridesep.ListDemandOverridesEndpoint{}).WithService(inner, svc)
	retrieveEndpoint := apiendpoint.From(&demandoverridesep.RetrieveDemandOverrideEndpoint{}).WithService(inner, svc)
	createEndpoint := apiendpoint.From(&demandoverridesep.CreateDemandOverrideEndpoint{}).WithService(inner, svc)
	updateEndpoint := apiendpoint.From(&demandoverridesep.UpdateDemandOverrideEndpoint{}).WithService(inner, svc)
	deleteEndpoint := apiendpoint.From(&demandoverridesep.DeleteDemandOverrideEndpoint{}).WithService(inner, svc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		listTypesEndpoint,
		listEndpoint,
		retrieveEndpoint,
		createEndpoint,
		updateEndpoint,
		deleteEndpoint,
	}

	return &DemandOverridesEndpointGroup{inner}
}
