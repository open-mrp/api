package httpgroup

import (
	"fmt"

	productep "github.com/augno/api/services/api-gateway/endpoints/products"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

type ProductsEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type ProductsEndpointGroupConfig struct {
	CoreClient *grpcclient.CoreServiceClient
}

func (c *ProductsEndpointGroupConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("products endpoint group: core client is required")
	}
	return nil
}

func (*ProductsEndpointGroup) Materialize(config *ProductsEndpointGroupConfig) *ProductsEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	productSvc := productep.NewProductSvc(&productep.ProductSvcConfig{
		CoreClient: config.CoreClient.Client,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Products Management",
		Description:  "List and manage products.",
		ResourceType: &apiresource.Product{},
	}

	listEndpoint := (&productep.ListProductsEndpoint{}).Materialize().WithService(inner, productSvc)
	retrieveEndpoint := (&productep.RetrieveProductEndpoint{}).Materialize().WithService(inner, productSvc)
	createEndpoint := (&productep.CreateProductEndpoint{}).Materialize().WithService(inner, productSvc)
	updateEndpoint := (&productep.UpdateProductEndpoint{}).Materialize().WithService(inner, productSvc)
	deleteEndpoint := (&productep.DeleteProductEndpoint{}).Materialize().WithService(inner, productSvc)
	changeProductLineEndpoint := (&productep.ChangeProductProductLineEndpoint{}).Materialize().WithService(inner, productSvc)
	validateEndpoint := (&productep.ValidateProductsEndpoint{}).Materialize().WithService(inner, productSvc)
	exportEndpoint := (&productep.ExportProductsEndpoint{}).Materialize().WithService(inner, productSvc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		listEndpoint,
		retrieveEndpoint,
		createEndpoint,
		updateEndpoint,
		deleteEndpoint,
		changeProductLineEndpoint,
		validateEndpoint,
		exportEndpoint,
	}

	return &ProductsEndpointGroup{inner}
}
