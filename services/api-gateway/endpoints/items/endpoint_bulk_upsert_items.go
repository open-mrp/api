package itemep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apirequest "github.com/open-mrp/api/services/api-gateway/pkg/request"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
	apiexample "github.com/open-mrp/apikit/example"
	"github.com/open-mrp/apikit/field"
	httptransport "github.com/open-mrp/apikit/transport"
)

// An attribute given by its property and value, such as `Size: Large`.
type UpsertItemAttribute struct {
	// Property name, matched case-insensitively. A property that doesn't exist is created and added to the item's category.
	Property string `json:"property" validate:"required,max=255"`
	// Attribute value within the property, matched case-insensitively. A value that doesn't exist is created.
	Value string `json:"value" validate:"required,max=255"`
}

// One item to create or update, matched to an existing item by `sku`.
type UpsertItemInput struct {
	// What kind of item this is: `material`, `part`, `product` or `service`. Must match an existing item's type.
	Type constants.ItemTypeCode `json:"type" validate:"required"`
	// Your stock keeping unit, unique (case-insensitive) within your account. Matches the row to an existing item.
	SKU string `json:"sku" validate:"required,max=255"`
	// Description of the item.
	Description field.Optional[string] `json:"description,omitzero" validate:"omitempty,max=5000"`
	// ID of the category the item is classified under.
	CategoryID string `json:"category_id" validate:"required"`
	// The item's attributes by property and value, at most one per property. Replaces the item's attributes when sent.
	Attributes field.Optional[[]UpsertItemAttribute] `json:"attributes,omitzero" validate:"omitempty,dive"`
	// Whether the item can be added to new orders, production and purchasing: `active` or `archived`. New items default to `active`.
	Status field.Optional[constants.ArchiveStatus] `json:"status,omitzero"`
	// Where the item comes from: `stock`, `make` or `buy`. Send `null` to inherit.
	Supply field.Clearable[constants.Supply] `json:"supply,omitzero"`
	// What one stocking unit of the item costs you.
	UnitCost field.Optional[apirequest.UnitPriceInput] `json:"unit_cost,omitzero" sensitive:"cost"`
	// Selling details, merged into the item's.
	Sales field.Optional[UpdateItemSalesInput] `json:"sales,omitzero"`
	// Purchasing details, for materials only, merged into the item's.
	Material field.Optional[UpdateItemMaterialInput] `json:"material,omitzero"`
	// ID of the unit the item's inventory, costs and usage are kept in.
	//
	// Required when the row creates a material, part or product. An existing item's must match; to change it, use Change Stocking Unit.
	StockingUnitID field.Optional[string] `json:"stocking_unit_id,omitzero"`
	// ID of the unit selected by default when the item is added to an order: the stocking unit or one of `order_units`.
	DefaultOrderUnitID field.Optional[string] `json:"default_order_unit_id,omitzero"`
	// Units the item can be ordered in besides its stocking unit. Replaces the item's list when sent.
	OrderUnits field.Optional[[]ItemOrderUnitInput] `json:"order_units,omitzero" validate:"omitempty,max=100,dive"`
}

// Request to create or update items in bulk.
type BulkUpsertItemsRequest struct {
	// Items to create or update, matched by SKU. At most 1,000.
	Items []UpsertItemInput `json:"items" validate:"required,min=1,max=1000,dive"`
}

var sampleBulkUpsertItemsRequest = &BulkUpsertItemsRequest{
	Items: []UpsertItemInput{
		{
			Type:           constants.ItemTypeCodeProduct,
			SKU:            apiresource.SampleItemSKU,
			CategoryID:     apiresource.SampleItemCategoryID,
			Attributes:     field.Some([]UpsertItemAttribute{{Property: "Size", Value: "Large"}}),
			StockingUnitID: field.Some(apiresource.SampleUnitID),
		},
	},
}

func (*BulkUpsertItemsRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleBulkUpsertItemsRequest)
}

// Creates or updates items in bulk.
//
// Rows are matched to existing items by `sku` (case-insensitive) and take the same fields as Create Item, sections by type. Attributes are given by property and value; missing ones are created. The work runs in the background; the response is an async job to poll.
type BulkUpsertItemsEndpoint struct{}

func (e *BulkUpsertItemsEndpoint) Materialize() *apiendpoint.APIEndpoint[*BulkUpsertItemsRequest, *apiresource.AsyncJob] {
	return (&apiendpoint.APIEndpoint[*BulkUpsertItemsRequest, *apiresource.AsyncJob]{
		Title:             "Bulk Upsert Items",
		Method:            http.MethodPost,
		Route:             "/v1/catalog/items/actions/bulk-upsert",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusAccepted,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainItems, Action: types.ActionCreate}, {Domain: types.PermissionDomainItems, Action: types.ActionUpdate}}, RequireAll: true},
		ObjectType:        constants.ObjectTypeAsyncJob,
		Extras:            apiendpoint.APIEndpointExtras{MaxJSONBodyBytes: httptransport.MaxJSONBodyBytes},
		ServiceHandler: func(svc any) func(ctx context.Context, req *BulkUpsertItemsRequest) (*apiresource.AsyncJob, *apierror.APIError) {
			return svc.(ItemSvc).BulkUpsertItems
		},
		LocationFunc: func(resp *apiresource.AsyncJob) string {
			return "/v1/core/async-jobs/" + resp.ID
		},
	})
}
