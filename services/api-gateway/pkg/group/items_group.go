package httpgroup

import (
	"fmt"

	itemep "github.com/open-mrp/api/services/api-gateway/endpoints/items"
	grpcclient "github.com/open-mrp/api/services/api-gateway/grpc-client"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

type ItemsEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type ItemsEndpointGroupConfig struct {
	// CoreClient (required) is the core-service gRPC client.
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
		Title:        "Items",
		Description:  "Materials, parts, products and services you buy, make, stock or sell.",
		ResourceType: &apiresource.Item{},
	}

	listItemsEndpoint := apiendpoint.From(&itemep.ListItemsEndpoint{}).WithService(inner, itemSvc)
	getItemEndpoint := apiendpoint.From(&itemep.RetrieveItemEndpoint{}).WithService(inner, itemSvc)
	createItemEndpoint := apiendpoint.From(&itemep.CreateItemEndpoint{}).WithService(inner, itemSvc)
	updateItemEndpoint := apiendpoint.From(&itemep.UpdateItemEndpoint{}).WithService(inner, itemSvc)
	deleteItemEndpoint := apiendpoint.From(&itemep.DeleteItemEndpoint{}).WithService(inner, itemSvc)
	bulkUpsertItemsEndpoint := apiendpoint.From(&itemep.BulkUpsertItemsEndpoint{}).WithService(inner, itemSvc)
	setOrderUnitsEndpoint := apiendpoint.From(&itemep.SetOrderUnitsEndpoint{}).WithService(inner, itemSvc)
	changeStockingUnitEndpoint := apiendpoint.From(&itemep.ChangeStockingUnitEndpoint{}).WithService(inner, itemSvc)
	getItemInventoryEndpoint := apiendpoint.From(&itemep.RetrieveItemInventoryEndpoint{}).WithService(inner, itemSvc)
	getItemLotDefaultEndpoint := apiendpoint.From(&itemep.RetrieveItemLotDefaultEndpoint{}).WithService(inner, itemSvc)
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
		createItemEndpoint,
		updateItemEndpoint,
		deleteItemEndpoint,
		bulkUpsertItemsEndpoint,
		setOrderUnitsEndpoint,
		changeStockingUnitEndpoint,
		getItemInventoryEndpoint,
		getItemLotDefaultEndpoint,
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
