package httpgroup

import (
	"fmt"

	productionstepep "github.com/augno/api/services/api-gateway/endpoints/production-steps"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

type ProductionStepsEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type ProductionStepsEndpointGroupConfig struct {
	// CoreClient (required) is the core-service gRPC client.
	CoreClient *grpcclient.CoreServiceClient
}

func (c *ProductionStepsEndpointGroupConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("production steps endpoint group: core client is required")
	}
	return nil
}

func (*ProductionStepsEndpointGroup) Materialize(config *ProductionStepsEndpointGroupConfig) *ProductionStepsEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	productionStepSvc := productionstepep.NewProductionStepSvc(&productionstepep.ProductionStepSvcConfig{
		CoreClient: config.CoreClient.ProductionStep,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Production Steps",
		Description:  "Manage production steps, their rates, productions, and consumptions.",
		ResourceType: &apiresource.ProductionStep{},
	}

	listEndpoint := apiendpoint.From(&productionstepep.ListProductionStepsEndpoint{}).WithService(inner, productionStepSvc)
	retrieveEndpoint := apiendpoint.From(&productionstepep.RetrieveProductionStepEndpoint{}).WithService(inner, productionStepSvc)
	createEndpoint := apiendpoint.From(&productionstepep.CreateProductionStepEndpoint{}).WithService(inner, productionStepSvc)
	updateEndpoint := apiendpoint.From(&productionstepep.UpdateProductionStepEndpoint{}).WithService(inner, productionStepSvc)
	deleteEndpoint := apiendpoint.From(&productionstepep.DeleteProductionStepEndpoint{}).WithService(inner, productionStepSvc)
	getProductionEndpoint := apiendpoint.From(&productionstepep.RetrieveProductionEndpoint{}).WithService(inner, productionStepSvc)
	updateProductionEndpoint := apiendpoint.From(&productionstepep.UpdateProductionEndpoint{}).WithService(inner, productionStepSvc)
	bulkCreateEndpoint := apiendpoint.From(&productionstepep.BulkCreateProductionStepsEndpoint{}).WithService(inner, productionStepSvc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		listEndpoint,
		retrieveEndpoint,
		createEndpoint,
		updateEndpoint,
		deleteEndpoint,
		getProductionEndpoint,
		updateProductionEndpoint,
		bulkCreateEndpoint,
	}

	return &ProductionStepsEndpointGroup{inner}
}
