package httpgroup

import (
	"fmt"

	catalogep "github.com/open-mrp/api/services/api-gateway/endpoints/catalog"
	grpcclient "github.com/open-mrp/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
)

type CatalogEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type CatalogEndpointGroupConfig struct {
	// CoreClient (required) is the core-service gRPC client.
	CoreClient *grpcclient.CoreServiceClient
}

func (c *CatalogEndpointGroupConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("catalog endpoint group: core client is required")
	}
	return nil
}

func (*CatalogEndpointGroup) Materialize(config *CatalogEndpointGroupConfig) *CatalogEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	catalogSvc := catalogep.NewCatalogSvc(&catalogep.CatalogSvcConfig{
		CoreClient: config.CoreClient.Client,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Catalog",
		Description:  "Browse the product catalog.",
		ResourceType: &apiresource.CatalogProductLine{},
	}

	listProductLinesEndpoint := apiendpoint.From(&catalogep.ListCatalogProductLinesEndpoint{}).WithService(inner, catalogSvc)
	listProductsEndpoint := apiendpoint.From(&catalogep.ListCatalogProductsEndpoint{}).WithService(inner, catalogSvc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		listProductLinesEndpoint,
		listProductsEndpoint,
	}

	return &CatalogEndpointGroup{inner}
}
