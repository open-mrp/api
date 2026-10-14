package purchaseorderep

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

// Request to update a purchase order line.
type UpdatePurchaseOrderLineRequest struct {
	// Purchase order ID.
	PurchaseOrderID string `path:"id" validate:"required"`
	// Purchase order line ID.
	PurchaseOrderLineID string `path:"line_id" validate:"required"`
	// ID of the product ordered on this line.
	ProductID field.Optional[string] `json:"product_id,omitzero" validate:"omitempty"`
	// ID of the inventory item to tie this line to.
	ItemID field.Optional[string] `json:"item_id,omitzero" validate:"omitempty"`
	// SKU of the ordered product.
	ProductSKU field.Optional[string] `json:"product_sku,omitzero" validate:"omitempty,max=255"`
	// Free-text description of the ordered product.
	ProductDescription field.Optional[string] `json:"product_description,omitzero"`
	// Quantity ordered, as a decimal string.
	QuantityValue field.Optional[string] `json:"quantity_value,omitzero" format:"decimal"`
	// ID of the unit the quantity is measured in.
	QuantityUnitID field.Optional[string] `json:"quantity_unit_id,omitzero" validate:"omitempty"`
	// Purchase price per unit.
	UnitPrice field.Optional[apirequest.UnitPriceInput] `json:"unit_price,omitzero"`
}

var sampleUpdatePOLineProductID = apiresource.SampleProductID
var sampleUpdatePOLineQuantityValue = "250"
var sampleUpdatePOLineProductSKU = "RAW-100"
var sampleUpdatePurchaseOrderLineRequest = &UpdatePurchaseOrderLineRequest{
	ProductID:     field.Some(sampleUpdatePOLineProductID),
	ProductSKU:    field.Some(sampleUpdatePOLineProductSKU),
	QuantityValue: field.Some(sampleUpdatePOLineQuantityValue),
	UnitPrice:     field.Some(apirequest.UnitPriceInput{Amount: "15.00", Currency: "usd", PerUnitID: apiresource.SampleUnitID}),
}

func (*UpdatePurchaseOrderLineRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdatePurchaseOrderLineRequest)
}

// Partially updates a purchase order line item.
//
// If the order has already been issued and the line has no open receiving order line, a new one is added for the quantity still outstanding, so an increased order quantity can be received.
//
// A line that references an inventory item also links that item's material to the supplier, if it is not linked already, so the material shows up as sourced from them.
type UpdatePurchaseOrderLineEndpoint struct{}

func (e *UpdatePurchaseOrderLineEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdatePurchaseOrderLineRequest, *apiresource.PurchaseOrderLine] {
	return (&apiendpoint.APIEndpoint[*UpdatePurchaseOrderLineRequest, *apiresource.PurchaseOrderLine]{
		Title:             "Update Purchase Order Line",
		Method:            http.MethodPatch,
		ContentType:       "application/json",
		Route:             "/v1/operations/purchase-orders/{id}/lines/{line_id}",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		AgentTool:         true,
		Preview:           true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainPurchaseOrders, Action: types.ActionUpdate}}, Counterparty: authz.CounterpartyPermissions{Supplier: types.Permission{Domain: types.PermissionDomainSuppliers, Action: types.ActionUpdate}}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdatePurchaseOrderLineRequest) (*apiresource.PurchaseOrderLine, *apierror.APIError) {
			return svc.(PurchaseOrderSvc).UpdatePurchaseOrderLine
		},
		ObjectType: constants.ObjectTypePurchaseOrderLine,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypePurchaseOrderLine,
			Fields:     purchaseOrderLineIncludeFields,
		}),
	})
}
