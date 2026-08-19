package productep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apirequest "github.com/augno/api/services/api-gateway/pkg/request"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// Request to create a product.
type CreateProductRequest struct {
	// Stock keeping unit code for the product's item.
	//
	// Must be unique within the account; creation fails with a conflict error if another item already uses it.
	SKU string `json:"sku" validate:"required,max=255"`
	// Free-form description of the product.
	Description field.Optional[string] `json:"description,omitzero"`
	// Free-form notes about the product.
	Notes field.Optional[string] `json:"notes,omitzero"`
	// Product type code, which determines how the product behaves on orders and invoices.
	//
	// - `sale`: a standard sellable product.
	// - `service`: a non-physical service line, such as labor or installation.
	// - `shipping`: a shipping charge applied to an order.
	// - `credit`: a credit applied against an order or invoice.
	// - `return`: a returned product (RMA).
	// - `tax`: a tax line.
	ProductTypeCode constants.ProductTypeCode `json:"type" validate:"required"`
	// ID of the product line to assign the product to.
	//
	// The product line must be one your account owns or a shared system line; anything else fails as not found. Buyers are granted access to whole product lines, so a product created without one never appears in the customer portal, whatever its `portal_visibility`.
	ProductLineID field.Optional[string] `json:"product_line_id,omitzero" validate:"omitempty"`
	// ID of the item category for the product's item.
	//
	// The category's unit group determines the default units used for the product's pricing rates and inventory tracking.
	CategoryID string `json:"category_id" validate:"required"`
	// Whether the product is shown to buyers in the customer portal.
	//
	// - `visible`: buyers can see and order the product in the portal.
	// - `hidden`: the product is concealed from the portal but remains usable internally.
	//
	// When omitted, the product is created hidden, so it must be set to `visible` before buyers can see it.
	PortalVisibility field.Optional[constants.CustomerPortalVisibility] `json:"portal_visibility,omitzero" default:"hidden"`
	// Initial selling price per unit.
	//
	// When set, the numerator unit must be a currency unit and the denominator unit must not be. When omitted, the price is initialized to a zero rate in the category's base unit.
	UnitPrice field.Optional[apirequest.RateInput] `json:"unit_price,omitzero"`
	// Initial cost per unit.
	//
	// The same unit rule as `unit_price` applies: the numerator unit must be a currency unit and the denominator unit must not be. When omitted, the cost is initialized to a zero rate in the category's base unit.
	UnitCost field.Optional[apirequest.RateInput] `json:"unit_cost,omitzero"`
	// Attribute IDs to link to the product's item at creation time.
	//
	// Every ID must already exist in your account, and each attribute's property must be one the item's category carries; an ID that fails either check fails the whole request rather than being skipped.
	AttributeIDs []string `json:"attribute_ids,omitzero"`
}

var sampleCreateProductDescription = "Wireless barcode scanner with charging cradle"
var sampleCreateProductNotes = "Ships with a 2-year warranty; register for extended coverage."
var sampleCreateProductRequest = &CreateProductRequest{
	SKU:              apiresource.SampleItemSKU,
	Description:      field.Some(sampleCreateProductDescription),
	Notes:            field.Some(sampleCreateProductNotes),
	ProductTypeCode:  apiresource.SampleProductTypeCode,
	ProductLineID:    field.Some(apiresource.SampleProductLineID),
	CategoryID:       apiresource.SampleItemCategoryID,
	PortalVisibility: field.Some(constants.CustomerPortalVisibilityVisible),
	UnitPrice:        field.Some(apirequest.RateInput{Value: "199.00", NumeratorUnitID: apiresource.SampleUnitID, DenominatorUnitID: apiresource.SampleUnitID}),
	UnitCost:         field.Some(apirequest.RateInput{Value: "112.00", NumeratorUnitID: apiresource.SampleUnitID, DenominatorUnitID: apiresource.SampleUnitID}),
	AttributeIDs:     []string{apiresource.SampleAttributeID},
}

func (*CreateProductRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateProductRequest)
}

// Creates a product and its backing inventory item.
//
// The new item starts with zero on-hand inventory, and its pricing defaults to zero rates in the category's base unit unless `unit_price` or `unit_cost` is provided.
//
// Only products of type `sale` appear in the product list and export; products created with any other type are still usable on orders and invoices but must be retrieved by ID.
type CreateProductEndpoint struct{}

func (e *CreateProductEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateProductRequest, *apiresource.Product] {
	return (&apiendpoint.APIEndpoint[*CreateProductRequest, *apiresource.Product]{
		Title:               "Create Product",
		Method:              http.MethodPost,
		ContentType:         "application/json",
		Route:               "/v1/catalog/products",
		SuccessStatusCode:   http.StatusCreated,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainItems, Action: types.ActionCreate}},
		Preview:             true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateProductRequest) (*apiresource.Product, *apierror.APIError) {
			return svc.(ProductSvc).CreateProduct
		},
		LocationFunc: func(resp *apiresource.Product) string {
			return "/v1/catalog/products/" + resp.ID
		},
		ObjectType: constants.ObjectTypeProduct,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeProduct,
			Fields:     []string{"product_line", "product_line.unit_group", "product_line.unit_group.base_unit", "product_line.unit_group.associated_units", "product_line.unit_group.associated_units.unit", "item", "item.category", "item.category.properties", "item.category.unit_group", "item.category.unit_group.base_unit", "item.category.unit_group.associated_units", "item.category.unit_group.associated_units.unit", "item.unit_value", "item.unit_cost", "item.burn_rate", "item.attributes"},
		}),
	})
}
