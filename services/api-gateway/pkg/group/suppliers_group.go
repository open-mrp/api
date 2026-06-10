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
	// CoreClient (required) is the core-service gRPC client.
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

	listEndpoint := apiendpoint.From(&supplierep.ListSuppliersEndpoint{}).WithService(inner, supplierSvc)
	retrieveEndpoint := apiendpoint.From(&supplierep.RetrieveSupplierEndpoint{}).WithService(inner, supplierSvc)
	createEndpoint := apiendpoint.From(&supplierep.CreateSupplierEndpoint{}).WithService(inner, supplierSvc)
	updateEndpoint := apiendpoint.From(&supplierep.UpdateSupplierEndpoint{}).WithService(inner, supplierSvc)
	deleteEndpoint := apiendpoint.From(&supplierep.DeleteSupplierEndpoint{}).WithService(inner, supplierSvc)
	bulkDeleteEndpoint := apiendpoint.From(&supplierep.BulkDeleteSuppliersEndpoint{}).WithService(inner, supplierSvc)

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
