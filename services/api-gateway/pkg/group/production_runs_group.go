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
		ResourceType: &apiresource.ProductionRunDetail{},
	}

	listEndpoint := (&productionrunep.ListProductionRunsEndpoint{}).Materialize().WithService(inner, svc)
	retrieveEndpoint := (&productionrunep.RetrieveProductionRunEndpoint{}).Materialize().WithService(inner, svc)
	createEndpoint := (&productionrunep.CreateProductionRunEndpoint{}).Materialize().WithService(inner, svc)
	updateEndpoint := (&productionrunep.UpdateProductionRunEndpoint{}).Materialize().WithService(inner, svc)
	deleteEndpoint := (&productionrunep.DeleteProductionRunEndpoint{}).Materialize().WithService(inner, svc)
	addBatchesEndpoint := (&productionrunep.AddBatchesToProductionRunEndpoint{}).Materialize().WithService(inner, svc)
	listBatchesEndpoint := (&productionrunep.ListBatchesByProductionRunEndpoint{}).Materialize().WithService(inner, svc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		listEndpoint,
		retrieveEndpoint,
		createEndpoint,
		updateEndpoint,
		deleteEndpoint,
		addBatchesEndpoint,
		listBatchesEndpoint,
	}

	return &ProductionRunsEndpointGroup{inner}
}
