package itemep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/field"
)

// One item to create in a bulk create request.
type BulkCreateItemInput struct {
	// SKU for the new item, unique within the account.
	//
	// If an item with this SKU already exists, that item is updated in place (description, category, and product line) instead of a new item being created; this path additionally requires permission to update items, and the result for the row is still reported with status `created`.
	SKU string `json:"sku" validate:"required"`
	// Item description.
	Description field.Optional[string] `json:"description,omitzero"`
	// ID of the category to assign to the item.
	//
	// The category determines the base unit the item's rates are expressed in, so choose one whose unit group matches how the item is counted.
	ItemCategoryID string `json:"item_category_id" validate:"required"`
	// ID of the product line to assign the item's product to.
	//
	// Only applies when `type` is `product`; ignored for materials and parts.
	ProductLineID field.Optional[string] `json:"product_line_id,omitzero"`
}

// Request to create multiple items of the same type.
type BulkCreateItemsRequest struct {
	// Items to create.
	Items []BulkCreateItemInput `json:"items" validate:"required"`
	// The item type applied to every item in the request.
	//
	// - `product`: a finished product.
	// - `material`: a raw material or component consumed in production.
	// - `part`: a part used in production.
	Type constants.ItemTypeCode `json:"type" validate:"required"`
}

var sampleBulkCreateItemsRequest = &BulkCreateItemsRequest{
	Type: constants.ItemTypeCodeMaterial,
	Items: []BulkCreateItemInput{
		{
			SKU:            "ALM-FLOUR-25LB",
			Description:    field.Some("Raw almond flour, 25 lb bag"),
			ItemCategoryID: apiresource.SampleItemCategoryID,
		},
	},
}

func (*BulkCreateItemsRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleBulkCreateItemsRequest)
}

// Creates multiple items of a single type in one call, returning a per-item result indicating success or failure.
//
// An input whose SKU already exists updates the existing item in place instead of creating a duplicate. A failure on one item does not abort the rest of the batch; check each result's status.
//
// Newly created items start with a unit value, unit cost, and burn rate of zero, counted in their category's base unit; set the real figures afterwards.
type BulkCreateItemsEndpoint struct{}

func (e *BulkCreateItemsEndpoint) Materialize() *apiendpoint.APIEndpoint[*BulkCreateItemsRequest, *apiresource.BulkCreateItemsResponse] {
	return (&apiendpoint.APIEndpoint[*BulkCreateItemsRequest, *apiresource.BulkCreateItemsResponse]{
		Title:               "Bulk Create Items",
		Method:              http.MethodPost,
		Route:               "/v1/catalog/items/actions/bulk-create",
		ContentType:         "application/json",
		SuccessStatusCode:   http.StatusCreated,
		Public:              false,
		Preview:             true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainItems, Action: types.ActionCreate}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *BulkCreateItemsRequest) (*apiresource.BulkCreateItemsResponse, *apierror.APIError) {
			return svc.(ItemSvc).BulkCreateItems
		},
	})
}
