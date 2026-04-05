package httpgroup

import (
	"fmt"

	catalogep "github.com/augno/api/services/api-gateway/endpoints/catalog"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

type CatalogEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type CatalogEndpointGroupConfig struct {
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

	listProductLinesEndpoint := (&catalogep.ListCatalogProductLinesEndpoint{}).Materialize().WithService(inner, catalogSvc)
	listProductsEndpoint := (&catalogep.ListCatalogProductsEndpoint{}).Materialize().WithService(inner, catalogSvc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		listProductLinesEndpoint,
		listProductsEndpoint,
	}

	return &CatalogEndpointGroup{inner}
}
