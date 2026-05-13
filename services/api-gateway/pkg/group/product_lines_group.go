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

	listEndpoint := (&productlineep.ListProductLinesEndpoint{}).Materialize().WithService(inner, productLineSvc)
	retrieveEndpoint := (&productlineep.RetrieveProductLineEndpoint{}).Materialize().WithService(inner, productLineSvc)
	createEndpoint := (&productlineep.CreateProductLineEndpoint{}).Materialize().WithService(inner, productLineSvc)
	updateEndpoint := (&productlineep.UpdateProductLineEndpoint{}).Materialize().WithService(inner, productLineSvc)
	deleteEndpoint := (&productlineep.DeleteProductLineEndpoint{}).Materialize().WithService(inner, productLineSvc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		listEndpoint,
		retrieveEndpoint,
		createEndpoint,
		updateEndpoint,
		deleteEndpoint,
	}

	return &ProductLinesEndpointGroup{inner}
}
