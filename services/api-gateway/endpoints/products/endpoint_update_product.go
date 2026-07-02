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

// UpdateProductRequest is the request to partially update a product.
type UpdateProductRequest struct {
	// Product ID.
	ProductID string `path:"id" validate:"required"`
	// New stock keeping unit code for the product's item.
	//
	// Must be unique within the account; the update fails with a conflict error if another item already uses it.
	SKU field.Optional[string] `json:"sku,omitzero" validate:"omitempty,max=255"`
	// Free-form description of the product.
	//
	// Send `null` to clear.
	Description field.Clearable[string] `json:"description,omitzero"`
	// Free-form notes about the product.
	//
	// Send `null` to clear.
	Notes field.Clearable[string] `json:"notes,omitzero"`
	// Whether the product is shown to buyers in the customer portal.
	//
	// - `visible`: buyers can see and order the product in the portal.
	// - `hidden`: the product is concealed from the portal but remains usable internally.
	PortalVisibility field.Optional[constants.CustomerPortalVisibility] `json:"portal_visibility,omitzero"`
	// New selling price per unit.
	//
	// The numerator unit must be a currency unit and the denominator unit must not be.
	UnitPrice field.Optional[apirequest.RateInput] `json:"unit_price,omitzero"`
}

var sampleUpdateProductSKU = "SKU-002"
var sampleUpdateProductDescription = "Wireless barcode scanner with charging cradle (v2)"
var sampleUpdateProductNotes = "Firmware 2.1 improves Bluetooth pairing reliability."

var sampleUpdateProductRequest = &UpdateProductRequest{
	SKU:              field.Some(sampleUpdateProductSKU),
	Description:      field.Set(sampleUpdateProductDescription),
	Notes:            field.Set(sampleUpdateProductNotes),
	PortalVisibility: field.Some(constants.CustomerPortalVisibilityVisible),
	UnitPrice:        field.Some(apirequest.RateInput{Value: "219.00", NumeratorUnitID: apiresource.SampleUnitID, DenominatorUnitID: apiresource.SampleUnitID}),
}

func (*UpdateProductRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateProductRequest)
}

// Partially updates a product.
type UpdateProductEndpoint struct{}

func (e *UpdateProductEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateProductRequest, *apiresource.Product] {
	return (&apiendpoint.APIEndpoint[*UpdateProductRequest, *apiresource.Product]{
		Title:               "Update Product",
		Method:              http.MethodPatch,
		Route:               "/v1/catalog/products/{id}",
		ContentType:         "application/json",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainItems, Action: types.ActionUpdate}},
		Preview:             true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateProductRequest) (*apiresource.Product, *apierror.APIError) {
			return svc.(ProductSvc).UpdateProduct
		},
		ObjectType: constants.ObjectTypeProduct,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeProduct,
			Fields:     []string{"product_line", "product_line.unit_group", "product_line.unit_group.base_unit", "product_line.unit_group.associated_units", "product_line.unit_group.associated_units.unit", "item", "item.category", "item.category.properties", "item.category.unit_group", "item.category.unit_group.base_unit", "item.category.unit_group.associated_units", "item.category.unit_group.associated_units.unit", "item.unit_value", "item.unit_cost", "item.burn_rate", "item.attributes"},
		}),
	})
}
