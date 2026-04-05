package httpgroup

import (
	"fmt"

	itemep "github.com/augno/api/services/api-gateway/endpoints/items"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

type ItemsEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type ItemsEndpointGroupConfig struct {
	CoreClient *grpcclient.CoreServiceClient
}

func (c *ItemsEndpointGroupConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("items endpoint group: core client is required")
	}
	return nil
}

func (*ItemsEndpointGroup) Materialize(config *ItemsEndpointGroupConfig) *ItemsEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	itemSvc := itemep.NewItemSvc(&itemep.ItemSvcConfig{
		CoreClient: config.CoreClient.Client,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Items Management",
		Description:  "List and manage inventory items.",
		ResourceType: &apiresource.Item{},
	}

	listItemsEndpoint := (&itemep.ListItemsEndpoint{}).Materialize().WithService(inner, itemSvc)
	getItemEndpoint := (&itemep.GetItemEndpoint{}).Materialize().WithService(inner, itemSvc)
	getItemInventoryEndpoint := (&itemep.GetItemInventoryEndpoint{}).Materialize().WithService(inner, itemSvc)
	getItemCostsEndpoint := (&itemep.GetItemCostsEndpoint{}).Materialize().WithService(inner, itemSvc)
	getItemTrendsEndpoint := (&itemep.GetItemTrendsEndpoint{}).Materialize().WithService(inner, itemSvc)
	exportItemsEndpoint := (&itemep.ExportItemsEndpoint{}).Materialize().WithService(inner, itemSvc)
	updateItemInventoryEndpoint := (&itemep.UpdateItemInventoryEndpoint{}).Materialize().WithService(inner, itemSvc)
	bulkCreateItemsEndpoint := (&itemep.BulkCreateItemsEndpoint{}).Materialize().WithService(inner, itemSvc)
	bulkReconcileItemsEndpoint := (&itemep.BulkReconcileItemsEndpoint{}).Materialize().WithService(inner, itemSvc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		listItemsEndpoint,
		getItemEndpoint,
		getItemInventoryEndpoint,
		getItemCostsEndpoint,
		getItemTrendsEndpoint,
		exportItemsEndpoint,
		updateItemInventoryEndpoint,
		bulkCreateItemsEndpoint,
		bulkReconcileItemsEndpoint,
	}

	return &ItemsEndpointGroup{inner}
}
