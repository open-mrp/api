package httpgroup

import (
	"fmt"

	producttypeep "github.com/augno/api/services/api-gateway/endpoints/product-types"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

type ProductTypesEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type ProductTypesEndpointGroupConfig struct {
	CoreClient *grpcclient.CoreServiceClient
}

func (c *ProductTypesEndpointGroupConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("product types endpoint group: core client is required")
	}
	return nil
}

func (*ProductTypesEndpointGroup) Materialize(config *ProductTypesEndpointGroupConfig) *ProductTypesEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	productTypeSvc := producttypeep.NewProductTypeSvc(&producttypeep.ProductTypeSvcConfig{
		CoreClient: config.CoreClient.Client,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Product Types Management",
		Description:  "List and manage product types.",
		ResourceType: &apiresource.ProductType{},
	}

	listProductTypesEndpoint := apiendpoint.From(&producttypeep.ListProductTypesEndpoint{}).WithService(inner, productTypeSvc)
	getProductTypeEndpoint := apiendpoint.From(&producttypeep.RetrieveProductTypeEndpoint{}).WithService(inner, productTypeSvc)
	createProductTypeEndpoint := apiendpoint.From(&producttypeep.CreateProductTypeEndpoint{}).WithService(inner, productTypeSvc)
	updateProductTypeEndpoint := apiendpoint.From(&producttypeep.UpdateProductTypeEndpoint{}).WithService(inner, productTypeSvc)
	deleteProductTypeEndpoint := apiendpoint.From(&producttypeep.DeleteProductTypeEndpoint{}).WithService(inner, productTypeSvc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		listProductTypesEndpoint,
		getProductTypeEndpoint,
		createProductTypeEndpoint,
		updateProductTypeEndpoint,
		deleteProductTypeEndpoint,
	}

	return &ProductTypesEndpointGroup{inner}
}
