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
	// CoreClient (required) is the core-service gRPC client.
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
		Title:        "Products",
		Description:  "List and manage products.",
		ResourceType: &apiresource.Product{},
	}

	listEndpoint := apiendpoint.From(&productep.ListProductsEndpoint{}).WithService(inner, productSvc)
	retrieveEndpoint := apiendpoint.From(&productep.RetrieveProductEndpoint{}).WithService(inner, productSvc)
	createEndpoint := apiendpoint.From(&productep.CreateProductEndpoint{}).WithService(inner, productSvc)
	updateEndpoint := apiendpoint.From(&productep.UpdateProductEndpoint{}).WithService(inner, productSvc)
	deleteEndpoint := apiendpoint.From(&productep.DeleteProductEndpoint{}).WithService(inner, productSvc)
	changeProductLineEndpoint := apiendpoint.From(&productep.ChangeProductProductLineEndpoint{}).WithService(inner, productSvc)
	validateEndpoint := apiendpoint.From(&productep.ValidateProductsEndpoint{}).WithService(inner, productSvc)
	exportEndpoint := apiendpoint.From(&productep.ExportProductsEndpoint{}).WithService(inner, productSvc)

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
