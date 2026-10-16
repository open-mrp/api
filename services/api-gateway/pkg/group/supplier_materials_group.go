package httpgroup

import (
	"fmt"

	supplieritemep "github.com/open-mrp/api/services/api-gateway/endpoints/supplier-items"
	suppliermaterialep "github.com/open-mrp/api/services/api-gateway/endpoints/supplier-materials"
	grpcclient "github.com/open-mrp/api/services/api-gateway/grpc-client"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

type SupplierMaterialsEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type SupplierMaterialsEndpointGroupConfig struct {
	// CoreClient (required) is the core-service gRPC client.
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
		Title:        "Supplier Materials",
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

type SupplierItemsEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

func (*SupplierItemsEndpointGroup) Materialize(config *SupplierMaterialsEndpointGroupConfig) *SupplierItemsEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	svc := supplieritemep.NewSupplierItemSvc(&supplieritemep.SupplierItemSvcConfig{CoreClient: config.CoreClient.Client})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Supplier Items",
		Description:  "The items each supplier provides, with the supplier's own codes for them.",
		ResourceType: &apiresource.SupplierItem{},
	}
	inner.Endpoints = []apiendpoint.APIEndpointer{
		apiendpoint.From(&supplieritemep.ListSupplierItemsEndpoint{}).WithService(inner, svc),
		apiendpoint.From(&supplieritemep.CreateSupplierItemEndpoint{}).WithService(inner, svc),
		apiendpoint.From(&supplieritemep.RetrieveSupplierItemEndpoint{}).WithService(inner, svc),
		apiendpoint.From(&supplieritemep.UpdateSupplierItemEndpoint{}).WithService(inner, svc),
		apiendpoint.From(&supplieritemep.DeleteSupplierItemEndpoint{}).WithService(inner, svc),
	}
	return &SupplierItemsEndpointGroup{inner}
}
