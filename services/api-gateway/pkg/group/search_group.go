package httpgroup

import (
	"fmt"

	searchep "github.com/open-mrp/api/services/api-gateway/endpoints/search"
	grpcclient "github.com/open-mrp/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
)

type SearchEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type SearchEndpointGroupConfig struct {
	// CoreClient (required) is the core-service gRPC client; its sub-clients back the sales-order, purchase-order, invoice, customer, item, product, and shipment searches.
	CoreClient *grpcclient.CoreServiceClient
	// NotificationClient (required) backs the messaging-contact search.
	NotificationClient *grpcclient.NotificationServiceClient
	// AgentClient (required) backs the agent search.
	AgentClient *grpcclient.AgentServiceClient
}

func (c *SearchEndpointGroupConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("search endpoint group: core client is required")
	}
	if c.NotificationClient == nil {
		return fmt.Errorf("search endpoint group: notification client is required")
	}
	if c.AgentClient == nil {
		return fmt.Errorf("search endpoint group: agent client is required")
	}
	return nil
}

func (*SearchEndpointGroup) Materialize(config *SearchEndpointGroupConfig) *SearchEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	svc := searchep.NewSearchSvc(&searchep.SearchSvcConfig{
		CoreClient:     config.CoreClient.Client,
		SalesClient:    config.CoreClient.Sales,
		PurchaseClient: config.CoreClient.Purchase,
		ShippingClient: config.CoreClient.Shipping,
		ChatClient:     config.NotificationClient.ChatClient,
		AgentClient:    config.AgentClient.Client,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Search",
		Description:  "Unified free-text search across resource types, returning lightweight entity references.",
		ResourceType: &apiresource.Entity{},
	}

	inner.Endpoints = []apiendpoint.APIEndpointer{
		apiendpoint.From(&searchep.SearchEndpoint{}).WithService(inner, svc),
	}

	return &SearchEndpointGroup{inner}
}
