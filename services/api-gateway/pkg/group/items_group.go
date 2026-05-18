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

	listItemsEndpoint := apiendpoint.From(&itemep.ListItemsEndpoint{}).WithService(inner, itemSvc)
	getItemEndpoint := apiendpoint.From(&itemep.RetrieveItemEndpoint{}).WithService(inner, itemSvc)
	getItemInventoryEndpoint := apiendpoint.From(&itemep.RetrieveItemInventoryEndpoint{}).WithService(inner, itemSvc)
	getItemCostsEndpoint := apiendpoint.From(&itemep.GetItemCostsEndpoint{}).WithService(inner, itemSvc)
	getItemTrendsEndpoint := apiendpoint.From(&itemep.GetItemTrendsEndpoint{}).WithService(inner, itemSvc)
	exportItemsEndpoint := apiendpoint.From(&itemep.ExportItemsEndpoint{}).WithService(inner, itemSvc)
	addItemAttributeEndpoint := apiendpoint.From(&itemep.AddItemAttributeEndpoint{}).WithService(inner, itemSvc)
	removeItemAttributeEndpoint := apiendpoint.From(&itemep.RemoveItemAttributeEndpoint{}).WithService(inner, itemSvc)
	changeItemCategoryEndpoint := apiendpoint.From(&itemep.ChangeItemCategoryEndpoint{}).WithService(inner, itemSvc)
	updateItemInventoryEndpoint := apiendpoint.From(&itemep.UpdateItemInventoryEndpoint{}).WithService(inner, itemSvc)
	bulkCreateItemsEndpoint := apiendpoint.From(&itemep.BulkCreateItemsEndpoint{}).WithService(inner, itemSvc)
	bulkReconcileItemsEndpoint := apiendpoint.From(&itemep.BulkReconcileItemsEndpoint{}).WithService(inner, itemSvc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		listItemsEndpoint,
		getItemEndpoint,
		getItemInventoryEndpoint,
		getItemCostsEndpoint,
		getItemTrendsEndpoint,
		exportItemsEndpoint,
		addItemAttributeEndpoint,
		removeItemAttributeEndpoint,
		changeItemCategoryEndpoint,
		updateItemInventoryEndpoint,
		bulkCreateItemsEndpoint,
		bulkReconcileItemsEndpoint,
	}

	return &ItemsEndpointGroup{inner}
}
