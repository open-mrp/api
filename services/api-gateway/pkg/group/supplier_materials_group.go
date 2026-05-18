package httpgroup

import (
	"fmt"

	suppliermaterialep "github.com/augno/api/services/api-gateway/endpoints/supplier-materials"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

type SupplierMaterialsEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type SupplierMaterialsEndpointGroupConfig struct {
	CoreClient *grpcclient.CoreServiceClient
}

func (c *SupplierMaterialsEndpointGroupConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("supplier materials endpoint group: core client is required")
	}
	return nil
}

func (*SupplierMaterialsEndpointGroup) Materialize(config *SupplierMaterialsEndpointGroupConfig) *SupplierMaterialsEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	supplierMaterialSvc := suppliermaterialep.NewSupplierMaterialSvc(&suppliermaterialep.SupplierMaterialSvcConfig{
		CoreClient: config.CoreClient.Client,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Supplier Materials Management",
		Description:  "List and manage supplier material associations.",
		ResourceType: &apiresource.SupplierMaterial{},
	}

	listEndpoint := apiendpoint.From(&suppliermaterialep.ListSupplierMaterialsEndpoint{}).WithService(inner, supplierMaterialSvc)
	retrieveEndpoint := apiendpoint.From(&suppliermaterialep.RetrieveSupplierMaterialEndpoint{}).WithService(inner, supplierMaterialSvc)
	createEndpoint := apiendpoint.From(&suppliermaterialep.CreateSupplierMaterialEndpoint{}).WithService(inner, supplierMaterialSvc)
	updateEndpoint := apiendpoint.From(&suppliermaterialep.UpdateSupplierMaterialEndpoint{}).WithService(inner, supplierMaterialSvc)
	deleteEndpoint := apiendpoint.From(&suppliermaterialep.DeleteSupplierMaterialEndpoint{}).WithService(inner, supplierMaterialSvc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		listEndpoint,
		retrieveEndpoint,
		createEndpoint,
		updateEndpoint,
		deleteEndpoint,
	}

	return &SupplierMaterialsEndpointGroup{inner}
}
