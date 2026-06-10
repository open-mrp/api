package httpgroup

import (
	"fmt"

	productlineep "github.com/augno/api/services/api-gateway/endpoints/product-lines"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

type ProductLinesEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type ProductLinesEndpointGroupConfig struct {
	// CoreClient (required) is the core-service gRPC client.
	CoreClient *grpcclient.CoreServiceClient
}

func (c *ProductLinesEndpointGroupConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("product lines endpoint group: core client is required")
	}
	return nil
}

func (*ProductLinesEndpointGroup) Materialize(config *ProductLinesEndpointGroupConfig) *ProductLinesEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	productLineSvc := productlineep.NewProductLineSvc(&productlineep.ProductLineSvcConfig{
		CoreClient: config.CoreClient.Client,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Product Lines Management",
		Description:  "List and manage product lines.",
		ResourceType: &apiresource.ProductLine{},
	}

	listEndpoint := apiendpoint.From(&productlineep.ListProductLinesEndpoint{}).WithService(inner, productLineSvc)
	retrieveEndpoint := apiendpoint.From(&productlineep.RetrieveProductLineEndpoint{}).WithService(inner, productLineSvc)
	createEndpoint := apiendpoint.From(&productlineep.CreateProductLineEndpoint{}).WithService(inner, productLineSvc)
	updateEndpoint := apiendpoint.From(&productlineep.UpdateProductLineEndpoint{}).WithService(inner, productLineSvc)
	deleteEndpoint := apiendpoint.From(&productlineep.DeleteProductLineEndpoint{}).WithService(inner, productLineSvc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		listEndpoint,
		retrieveEndpoint,
		createEndpoint,
		updateEndpoint,
		deleteEndpoint,
	}

	return &ProductLinesEndpointGroup{inner}
}
