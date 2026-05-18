package itemep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// BulkCreateItemInput is the input for a single item in a bulk create operation.
type BulkCreateItemInput struct {
	// Item SKU.
	SKU string `json:"sku" validate:"required"`
	// Item description.
	Description *string `json:"description,omitempty"`
	// Item category ID.
	ItemCategoryID string `json:"item_category_id" validate:"required"`
	// Product line ID.
	ProductLineID *string `json:"product_line_id,omitempty"`
}

// BulkCreateItemsRequest is the request to create multiple items.
type BulkCreateItemsRequest struct {
	// Items to create.
	Items []BulkCreateItemInput `json:"items" validate:"required"`
	// Item type (product, material, or part).
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

// Creates multiple items in a single operation, returning per-item results indicating success or failure.
type BulkCreateItemsEndpoint struct{}

func (e *BulkCreateItemsEndpoint) Materialize() *apiendpoint.APIEndpoint[*BulkCreateItemsRequest, *apiresource.BulkCreateItemsResponse] {
	return (&apiendpoint.APIEndpoint[*BulkCreateItemsRequest, *apiresource.BulkCreateItemsResponse]{
		Title:             "Bulk Create Items",
		Method:            http.MethodPost,
		Route:             "/v1/catalog/items/actions/bulk-create",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusCreated,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *BulkCreateItemsRequest) (*apiresource.BulkCreateItemsResponse, *apierror.APIError) {
			return svc.(ItemSvc).BulkCreateItems
		},
	})
}
