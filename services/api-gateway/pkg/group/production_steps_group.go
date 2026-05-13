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
		Title:        "Production Steps Management",
		Description:  "Manage production steps, their rates, productions, and consumptions.",
		ResourceType: &apiresource.ProductionStep{},
	}

	listEndpoint := (&productionstepep.ListProductionStepsEndpoint{}).Materialize().WithService(inner, productionStepSvc)
	retrieveEndpoint := (&productionstepep.RetrieveProductionStepEndpoint{}).Materialize().WithService(inner, productionStepSvc)
	createEndpoint := (&productionstepep.CreateProductionStepEndpoint{}).Materialize().WithService(inner, productionStepSvc)
	updateEndpoint := (&productionstepep.UpdateProductionStepEndpoint{}).Materialize().WithService(inner, productionStepSvc)
	deleteEndpoint := (&productionstepep.DeleteProductionStepEndpoint{}).Materialize().WithService(inner, productionStepSvc)
	getProductionEndpoint := (&productionstepep.RetrieveProductionEndpoint{}).Materialize().WithService(inner, productionStepSvc)
	updateProductionEndpoint := (&productionstepep.UpdateProductionEndpoint{}).Materialize().WithService(inner, productionStepSvc)
	bulkCreateEndpoint := (&productionstepep.BulkCreateProductionStepsEndpoint{}).Materialize().WithService(inner, productionStepSvc)

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
