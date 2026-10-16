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
)

// An item's selling details on update. Only fields you send are changed; a material or part without them gains them.
type UpdateItemSalesInput struct {
	// Base price on sales orders, before price lists apply.
	UnitPrice field.Optional[apirequest.UnitPriceInput] `json:"unit_price,omitzero"`
	// ID of the product line the item is sold under. Must be active. Send `null` to remove the item from its line.
	ProductLineID field.Clearable[string] `json:"product_line_id,omitzero"`
	// Whether customers with access to the item's product line can see and order it in the portal: `visible` or `hidden`.
	PortalVisibility field.Optional[constants.CustomerPortalVisibility] `json:"portal_visibility,omitzero"`
	// How long production or purchasing takes before the item can ship, in business days, in a time unit. Send `null` to remove it.
	LeadTime field.Clearable[apirequest.QuantityInput] `json:"lead_time,omitzero"`
}

// A material's purchasing details on update. Only fields you send are changed.
type UpdateItemMaterialInput struct {
	// Reorder when on-hand stock falls to this quantity. Send `null` to remove the order point.
	OrderPoint field.Clearable[apirequest.QuantityInput] `json:"order_point,omitzero"`
	// How long the material usually takes to arrive after it's ordered, in a time unit. Send `null` when unknown.
	LeadTime field.Clearable[apirequest.QuantityInput] `json:"lead_time,omitzero"`
}

// Request to update an item.
type UpdateItemRequest struct {
	// Item ID.
	ItemID string `path:"id" validate:"required"`
	// Your stock keeping unit, unique (case-insensitive) within your account.
	SKU field.Optional[string] `json:"sku,omitzero" validate:"omitempty,max=255"`
	// Description of the item. Send `null` to remove it.
	Description field.Clearable[string] `json:"description,omitzero" validate:"omitempty,max=5000"`
	// ID of the category the item is classified under, of the item's kind. It must carry the properties of the item's attributes.
	CategoryID field.Optional[string] `json:"category_id,omitzero"`
	// IDs of the item's attributes, at most one per property. Replaces the whole list.
	AttributeIDs field.Optional[[]string] `json:"attribute_ids,omitzero"`
	// Whether the item can be added to new orders, production and purchasing: `active` or `archived`.
	Status field.Optional[constants.ArchiveStatus] `json:"status,omitzero"`
	// Where the item comes from when an order line or plan doesn't say: `stock`, `make` or `buy`. Send `null` to use the product line's, then the account default.
	Supply field.Clearable[constants.Supply] `json:"supply,omitzero"`
	// What one stocking unit of the item costs you.
	UnitCost field.Optional[apirequest.UnitPriceInput] `json:"unit_cost,omitzero" sensitive:"cost"`
	// Selling details to change. Sending them for a material or part without a sales section makes it sellable. Send `null` to stop selling a material or part, refused while open sales orders list it.
	Sales field.Clearable[UpdateItemSalesInput] `json:"sales,omitzero"`
	// Purchasing details to change, for materials only.
	Material field.Optional[UpdateItemMaterialInput] `json:"material,omitzero"`
}

var sampleUpdateItemRequest = &UpdateItemRequest{
	Description: field.Set("6061-T6 aluminum sheet, 4ft x 8ft, 0.125in thick (v2)"),
	Status:      field.Some(constants.ArchiveStatusActive),
	Sales: field.Set(UpdateItemSalesInput{
		UnitPrice:     field.Some(apirequest.UnitPriceInput{Amount: "219.00", Currency: "usd", PerUnitID: apiresource.SampleUnitID}),
		ProductLineID: field.Set(apiresource.SampleProductLineID),
	}),
}

func (*UpdateItemRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateItemRequest)
}

// Updates an item.
//
// Only fields you send are changed. To change the stocking unit, use Change Stocking Unit.
type UpdateItemEndpoint struct{}

func (e *UpdateItemEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateItemRequest, *apiresource.Item] {
	return (&apiendpoint.APIEndpoint[*UpdateItemRequest, *apiresource.Item]{
		Title:             "Update Item",
		Method:            http.MethodPatch,
		ContentType:       "application/json",
		Route:             itemRoute,
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainItems, Action: types.ActionUpdate}}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateItemRequest) (*apiresource.Item, *apierror.APIError) {
			return svc.(ItemSvc).UpdateItem
		},
		ObjectType:    constants.ObjectTypeItem,
		IncludeConfig: itemIncludes(),
	})
}
