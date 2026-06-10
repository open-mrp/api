package httpgroup

import (
	"fmt"

	itemcategoryep "github.com/augno/api/services/api-gateway/endpoints/item-categories"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

type ItemCategoriesEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type ItemCategoriesEndpointGroupConfig struct {
	// CoreClient (required) is the core-service gRPC client.
	CoreClient *grpcclient.CoreServiceClient
}

func (c *ItemCategoriesEndpointGroupConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("item categories endpoint group: core client is required")
	}
	return nil
}

func (*ItemCategoriesEndpointGroup) Materialize(config *ItemCategoriesEndpointGroupConfig) *ItemCategoriesEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	itemCategorySvc := itemcategoryep.NewItemCategorySvc(&itemcategoryep.ItemCategorySvcConfig{
		CoreClient: config.CoreClient.Client,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Item Categories Management",
		Description:  "List and manage item categories.",
		ResourceType: &apiresource.ItemCategory{},
	}

	listEndpoint := apiendpoint.From(&itemcategoryep.ListItemCategoriesEndpoint{}).WithService(inner, itemCategorySvc)
	retrieveEndpoint := apiendpoint.From(&itemcategoryep.RetrieveItemCategoryEndpoint{}).WithService(inner, itemCategorySvc)
	createEndpoint := apiendpoint.From(&itemcategoryep.CreateItemCategoryEndpoint{}).WithService(inner, itemCategorySvc)
	updateEndpoint := apiendpoint.From(&itemcategoryep.UpdateItemCategoryEndpoint{}).WithService(inner, itemCategorySvc)
	deleteEndpoint := apiendpoint.From(&itemcategoryep.DeleteItemCategoryEndpoint{}).WithService(inner, itemCategorySvc)
	addPropertyEndpoint := apiendpoint.From(&itemcategoryep.AddItemCategoryPropertyEndpoint{}).WithService(inner, itemCategorySvc)
	removePropertyEndpoint := apiendpoint.From(&itemcategoryep.RemoveItemCategoryPropertyEndpoint{}).WithService(inner, itemCategorySvc)
	changeUnitGroupEndpoint := apiendpoint.From(&itemcategoryep.ChangeItemCategoryUnitGroupEndpoint{}).WithService(inner, itemCategorySvc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		listEndpoint,
		retrieveEndpoint,
		createEndpoint,
		updateEndpoint,
		deleteEndpoint,
		addPropertyEndpoint,
		removePropertyEndpoint,
		changeUnitGroupEndpoint,
	}

	return &ItemCategoriesEndpointGroup{inner}
}
