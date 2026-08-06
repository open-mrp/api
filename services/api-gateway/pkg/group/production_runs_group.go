package httpgroup

import (
	"fmt"

	productionrunep "github.com/augno/api/services/api-gateway/endpoints/production-runs"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

type ProductionRunsEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type ProductionRunsEndpointGroupConfig struct {
	// CoreClient (required) is the core-service gRPC client.
	CoreClient *grpcclient.CoreServiceClient
}

func (c *ProductionRunsEndpointGroupConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("production runs endpoint group: core client is required")
	}
	return nil
}

func (*ProductionRunsEndpointGroup) Materialize(config *ProductionRunsEndpointGroupConfig) *ProductionRunsEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	svc := productionrunep.NewProductionRunSvc(&productionrunep.ProductionRunSvcConfig{
		CoreClient: config.CoreClient.ProductionRun,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Production Runs",
		Description:  "List, view, create, update, and delete production runs.",
		ResourceType: &apiresource.ProductionRun{},
	}

	listEndpoint := apiendpoint.From(&productionrunep.ListProductionRunsEndpoint{}).WithService(inner, svc)
	retrieveEndpoint := apiendpoint.From(&productionrunep.RetrieveProductionRunEndpoint{}).WithService(inner, svc)
	createEndpoint := apiendpoint.From(&productionrunep.CreateProductionRunEndpoint{}).WithService(inner, svc)
	updateEndpoint := apiendpoint.From(&productionrunep.UpdateProductionRunEndpoint{}).WithService(inner, svc)
	deleteEndpoint := apiendpoint.From(&productionrunep.DeleteProductionRunEndpoint{}).WithService(inner, svc)
	addBatchesEndpoint := apiendpoint.From(&productionrunep.AddBatchesToProductionRunEndpoint{}).WithService(inner, svc)
	listBatchesEndpoint := apiendpoint.From(&productionrunep.ListBatchesByProductionRunEndpoint{}).WithService(inner, svc)
	bulkCreateEndpoint := apiendpoint.From(&productionrunep.BulkCreateProductionRunsEndpoint{}).WithService(inner, svc)
	exportEndpoint := apiendpoint.From(&productionrunep.ExportProductionRunsEndpoint{}).WithService(inner, svc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		listEndpoint,
		retrieveEndpoint,
		createEndpoint,
		updateEndpoint,
		deleteEndpoint,
		addBatchesEndpoint,
		listBatchesEndpoint,
		bulkCreateEndpoint,
		exportEndpoint,
	}

	return &ProductionRunsEndpointGroup{inner}
}
