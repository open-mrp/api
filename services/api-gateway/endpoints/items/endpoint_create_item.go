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

// An item's selling details on create.
type CreateItemSalesInput struct {
	// Base price on sales orders, before price lists apply. Defaults to zero per the category's base unit.
	UnitPrice field.Optional[apirequest.UnitPriceInput] `json:"unit_price,omitzero"`
	// ID of the product line the item is sold under. Must be active.
	ProductLineID field.Optional[string] `json:"product_line_id,omitzero"`
	// Whether customers with access to the item's product line can see and order it in the portal: `visible` or `hidden`.
	//
	// Defaults to `hidden`.
	PortalVisibility field.Optional[constants.CustomerPortalVisibility] `json:"portal_visibility,omitzero"`
	// How long production or purchasing takes before the item can ship, in business days, in a time unit.
	LeadTime field.Optional[apirequest.QuantityInput] `json:"lead_time,omitzero"`
}

// A material's purchasing details on create.
type CreateItemMaterialInput struct {
	// Reorder when on-hand stock falls to this quantity.
	OrderPoint field.Optional[apirequest.QuantityInput] `json:"order_point,omitzero"`
	// How long the material usually takes to arrive after it's ordered, in a time unit.
	LeadTime field.Optional[apirequest.QuantityInput] `json:"lead_time,omitzero"`
}

// Request to create an item.
type CreateItemRequest struct {
	// What kind of item this is: `material`, `part`, `product` or `service`. Can't change after creation.
	Type constants.ItemTypeCode `json:"type" validate:"required"`
	// Your stock keeping unit, unique (case-insensitive) within your account.
	SKU string `json:"sku" validate:"required,max=255"`
	// Description of the item.
	Description field.Optional[string] `json:"description,omitzero" validate:"omitempty,max=5000"`
	// ID of the category the item is classified under.
	//
	// A material needs a `material` category; parts, products and services need a `product` category.
	CategoryID string `json:"category_id" validate:"required"`
	// IDs of the item's attributes, at most one per property. Each attribute's property must be one the category carries, and the attribute must be active.
	AttributeIDs []string `json:"attribute_ids,omitzero"`
	// Whether the item can be added to new orders, production and purchasing: `active` or `archived`.
	//
	// Defaults to `active`.
	Status field.Optional[constants.ArchiveStatus] `json:"status,omitzero"`
	// Where the item comes from when an order line or plan doesn't say: `stock`, `make` or `buy`. Omit to use the product line's, then the account default.
	Supply field.Optional[constants.Supply] `json:"supply,omitzero"`
	// What one stocking unit of the item costs you. Defaults to zero per the category's base unit.
	UnitCost field.Optional[apirequest.UnitPriceInput] `json:"unit_cost,omitzero" sensitive:"cost"`
	// Selling details. Products and services always have them; send them for a material or part you sell.
	Sales field.Optional[CreateItemSalesInput] `json:"sales,omitzero"`
	// Purchasing details, for materials only.
	Material field.Optional[CreateItemMaterialInput] `json:"material,omitzero"`
}

var sampleCreateItemRequest = &CreateItemRequest{
	Type:         constants.ItemTypeCodeProduct,
	SKU:          apiresource.SampleItemSKU,
	Description:  field.Some("6061-T6 aluminum sheet, 4ft x 8ft, 0.125in thick"),
	CategoryID:   apiresource.SampleItemCategoryID,
	AttributeIDs: []string{apiresource.SampleAttributeID},
	UnitCost:     field.Some(apirequest.UnitPriceInput{Amount: "112.00", Currency: "usd", PerUnitID: apiresource.SampleUnitID}),
	Sales: field.Some(CreateItemSalesInput{
		UnitPrice:        field.Some(apirequest.UnitPriceInput{Amount: "199.00", Currency: "usd", PerUnitID: apiresource.SampleUnitID}),
		ProductLineID:    field.Some(apiresource.SampleProductLineID),
		PortalVisibility: field.Some(constants.CustomerPortalVisibilityVisible),
	}),
}

func (*CreateItemRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateItemRequest)
}

// Creates an item.
//
// `type` can't change later. Send the `sales` section for products and services, and `material` for materials.
type CreateItemEndpoint struct{}

func (e *CreateItemEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateItemRequest, *apiresource.Item] {
	return (&apiendpoint.APIEndpoint[*CreateItemRequest, *apiresource.Item]{
		Title:             "Create Item",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             itemsRoute,
		SuccessStatusCode: http.StatusCreated,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainItems, Action: types.ActionCreate}}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateItemRequest) (*apiresource.Item, *apierror.APIError) {
			return svc.(ItemSvc).CreateItem
		},
		LocationFunc: func(resp *apiresource.Item) string {
			return itemsRoute + "/" + resp.ID
		},
		ObjectType:    constants.ObjectTypeItem,
		IncludeConfig: itemIncludes(),
	})
}
