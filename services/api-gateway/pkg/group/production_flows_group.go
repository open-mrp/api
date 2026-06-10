package httpgroup

import (
	"fmt"

	productionflowep "github.com/augno/api/services/api-gateway/endpoints/production-flows"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

type ProductionFlowsEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type ProductionFlowsEndpointGroupConfig struct {
	// CoreClient (required) is the core-service gRPC client.
	CoreClient *grpcclient.CoreServiceClient
}

func (c *ProductionFlowsEndpointGroupConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("production flows endpoint group: core client is required")
	}
	return nil
}

func (*ProductionFlowsEndpointGroup) Materialize(config *ProductionFlowsEndpointGroupConfig) *ProductionFlowsEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	productionFlowSvc := productionflowep.NewProductionFlowSvc(&productionflowep.ProductionFlowSvcConfig{
		CoreClient: config.CoreClient.Client,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Production Flows",
		Description:  "Retrieve and manage production flow graphs.",
		ResourceType: &apiresource.ProductionFlow{},
	}

	getProductionFlowEndpoint := apiendpoint.From(&productionflowep.GetProductionFlowEndpoint{}).WithService(inner, productionFlowSvc)
	connectStepsEndpoint := apiendpoint.From(&productionflowep.ConnectStepsEndpoint{}).WithService(inner, productionFlowSvc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		getProductionFlowEndpoint,
		connectStepsEndpoint,
	}

	return &ProductionFlowsEndpointGroup{inner}
}
