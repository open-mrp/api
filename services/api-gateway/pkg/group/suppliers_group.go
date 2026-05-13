package httpgroup

import (
	"fmt"

	supplierep "github.com/augno/api/services/api-gateway/endpoints/suppliers"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

type SuppliersEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type SuppliersEndpointGroupConfig struct {
	CoreClient *grpcclient.CoreServiceClient
}

func (c *SuppliersEndpointGroupConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("suppliers endpoint group: core client is required")
	}
	return nil
}

func (*SuppliersEndpointGroup) Materialize(config *SuppliersEndpointGroupConfig) *SuppliersEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	supplierSvc := supplierep.NewSupplierSvc(&supplierep.SupplierSvcConfig{
		CoreClient: config.CoreClient.Client,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Suppliers Management",
		Description:  "List and manage suppliers.",
		ResourceType: &apiresource.SupplierDetail{},
	}

	listEndpoint := (&supplierep.ListSuppliersEndpoint{}).Materialize().WithService(inner, supplierSvc)
	retrieveEndpoint := (&supplierep.RetrieveSupplierEndpoint{}).Materialize().WithService(inner, supplierSvc)
	createEndpoint := (&supplierep.CreateSupplierEndpoint{}).Materialize().WithService(inner, supplierSvc)
	updateEndpoint := (&supplierep.UpdateSupplierEndpoint{}).Materialize().WithService(inner, supplierSvc)
	deleteEndpoint := (&supplierep.DeleteSupplierEndpoint{}).Materialize().WithService(inner, supplierSvc)
	bulkDeleteEndpoint := (&supplierep.BulkDeleteSuppliersEndpoint{}).Materialize().WithService(inner, supplierSvc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		listEndpoint,
		retrieveEndpoint,
		createEndpoint,
		updateEndpoint,
		deleteEndpoint,
		bulkDeleteEndpoint,
	}

	return &SuppliersEndpointGroup{inner}
}
