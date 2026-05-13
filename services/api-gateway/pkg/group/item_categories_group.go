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

	listEndpoint := (&itemcategoryep.ListItemCategoriesEndpoint{}).Materialize().WithService(inner, itemCategorySvc)
	retrieveEndpoint := (&itemcategoryep.RetrieveItemCategoryEndpoint{}).Materialize().WithService(inner, itemCategorySvc)
	createEndpoint := (&itemcategoryep.CreateItemCategoryEndpoint{}).Materialize().WithService(inner, itemCategorySvc)
	updateEndpoint := (&itemcategoryep.UpdateItemCategoryEndpoint{}).Materialize().WithService(inner, itemCategorySvc)
	deleteEndpoint := (&itemcategoryep.DeleteItemCategoryEndpoint{}).Materialize().WithService(inner, itemCategorySvc)
	addPropertyEndpoint := (&itemcategoryep.AddItemCategoryPropertyEndpoint{}).Materialize().WithService(inner, itemCategorySvc)
	removePropertyEndpoint := (&itemcategoryep.RemoveItemCategoryPropertyEndpoint{}).Materialize().WithService(inner, itemCategorySvc)
	changeUnitGroupEndpoint := (&itemcategoryep.ChangeItemCategoryUnitGroupEndpoint{}).Materialize().WithService(inner, itemCategorySvc)

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
