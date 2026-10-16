package httpgroup

import (
	"fmt"

	pricelistep "github.com/open-mrp/api/services/api-gateway/endpoints/price-lists"
	grpcclient "github.com/open-mrp/api/services/api-gateway/grpc-client"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

type PriceListsEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type PriceListsEndpointGroupConfig struct {
	// CoreClient (required) is the core-service gRPC client.
	CoreClient *grpcclient.CoreServiceClient
}

func (c *PriceListsEndpointGroupConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("price lists endpoint group: core client is required")
	}
	return nil
}

func (*PriceListsEndpointGroup) Materialize(config *PriceListsEndpointGroupConfig) *PriceListsEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	svc := pricelistep.NewPriceListSvc(&pricelistep.PriceListSvcConfig{
		CoreClient:   config.CoreClient.PriceList,
		ExportClient: config.CoreClient.Client,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Price Lists",
		Description:  "Pricing rules you assign to customers and account groups. Order lines take their price from the most specific matching rule.",
		ResourceType: &apiresource.PriceList{},
	}

	inner.Endpoints = []apiendpoint.APIEndpointer{
		apiendpoint.From(&pricelistep.ListPriceListsEndpoint{}).WithService(inner, svc),
		apiendpoint.From(&pricelistep.RetrievePriceListEndpoint{}).WithService(inner, svc),
		apiendpoint.From(&pricelistep.CreatePriceListEndpoint{}).WithService(inner, svc),
		apiendpoint.From(&pricelistep.UpdatePriceListEndpoint{}).WithService(inner, svc),
		apiendpoint.From(&pricelistep.DeletePriceListEndpoint{}).WithService(inner, svc),
		apiendpoint.From(&pricelistep.ListPriceListRulesEndpoint{}).WithService(inner, svc),
		apiendpoint.From(&pricelistep.RetrievePriceListRuleEndpoint{}).WithService(inner, svc),
		apiendpoint.From(&pricelistep.CreatePriceListRuleEndpoint{}).WithService(inner, svc),
		apiendpoint.From(&pricelistep.UpdatePriceListRuleEndpoint{}).WithService(inner, svc),
		apiendpoint.From(&pricelistep.DeletePriceListRuleEndpoint{}).WithService(inner, svc),
		apiendpoint.From(&pricelistep.ExportCustomerPricesEndpoint{}).WithService(inner, svc),
	}

	return &PriceListsEndpointGroup{inner}
}
