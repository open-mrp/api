package itemep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// BulkCreateItemInput represents a single item to create in a bulk operation.
type BulkCreateItemInput struct {
	// The SKU for the item.
	SKU string `json:"sku" validate:"required"`
	// The description for the item.
	Description *string `json:"description,omitempty"`
	// The item category ID.
	ItemCategoryID string `json:"item_category_id" validate:"required"`
	// The product line ID.
	ProductLineID *string `json:"product_line_id,omitempty"`
}

// BulkCreateItemsRequest is the request to create multiple items.
type BulkCreateItemsRequest struct {
	// The items to create.
	Items []BulkCreateItemInput `json:"items" validate:"required"`
	// The type of items to create (product, material, part).
	Type string `json:"type" validate:"required"`
}

var sampleBulkCreateItemDescription = "Raw almond flour, 25 lb bag"
var sampleBulkCreateItemsRequest = &BulkCreateItemsRequest{
	Type: "material",
	Items: []BulkCreateItemInput{
		{
			SKU:            "ALM-FLOUR-25LB",
			Description:    &sampleBulkCreateItemDescription,
			ItemCategoryID: apiresource.SampleItemCategoryID,
		},
	},
}

func (*BulkCreateItemsRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleBulkCreateItemsRequest)
}

type BulkCreateItemsEndpoint struct{}

func (e *BulkCreateItemsEndpoint) Materialize() *apiendpoint.APIEndpoint[*BulkCreateItemsRequest, *apiresource.BulkCreateItemsResponse] {
	return &apiendpoint.APIEndpoint[*BulkCreateItemsRequest, *apiresource.BulkCreateItemsResponse]{
		Title:             "Bulk Create Items",
		Description:       "Creates multiple items in a single operation, returning per-item results indicating success or failure.",
		Method:            http.MethodPost,
		Route:             "/v1/catalog/items/actions/bulk-create",
		ContentType:       "application/json",
		Request:           &BulkCreateItemsRequest{},
		Response:          &apiresource.BulkCreateItemsResponse{},
		SuccessStatusCode: http.StatusCreated,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *BulkCreateItemsRequest) (*apiresource.BulkCreateItemsResponse, *apierror.APIError) {
			return svc.(ItemSvc).BulkCreateItems
		},
	}
}
