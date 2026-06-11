package purchaseorderep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// Request to update a purchase order line.
type UpdatePurchaseOrderLineRequest struct {
	// Purchase order ID.
	PurchaseOrderID string `path:"id" validate:"required"`
	// Purchase order line ID.
	PurchaseOrderLineID string `path:"line_id" validate:"required"`
	// Product ID.
	ProductID field.Optional[string] `json:"product_id,omitzero" validate:"omitempty"`
	// ID of the catalog item to link this line to.
	ItemID field.Optional[string] `json:"item_id,omitzero" validate:"omitempty"`
	// SKU of the ordered product.
	ProductSKU field.Optional[string] `json:"product_sku,omitzero" validate:"omitempty,max=255"`
	// Free-text description of the ordered product.
	ProductDescription field.Optional[string] `json:"product_description,omitzero"`
	// Quantity ordered, as a decimal string.
	QuantityValue field.Optional[string] `json:"quantity_value,omitzero" format:"decimal"`
	// ID of the unit the quantity is measured in.
	QuantityUnitID field.Optional[string] `json:"quantity_unit_id,omitzero" validate:"omitempty"`
	// Purchase price per unit, as a decimal string.
	UnitPriceValue field.Optional[string] `json:"unit_price_value,omitzero" format:"decimal"`
	// ID of the unit price's numerator unit (e.g. a currency unit).
	UnitPriceNumeratorUnitID field.Optional[string] `json:"unit_price_numerator_unit_id,omitzero" validate:"omitempty"`
	// ID of the unit price's denominator unit (the unit the price is per).
	UnitPriceDenominatorUnitID field.Optional[string] `json:"unit_price_denominator_unit_id,omitzero" validate:"omitempty"`
	// Recorded cost per unit, as a decimal string.
	UnitCostValue field.Optional[string] `json:"unit_cost_value,omitzero" format:"decimal"`
	// ID of the unit cost's numerator unit (e.g. a currency unit).
	UnitCostNumeratorUnitID field.Optional[string] `json:"unit_cost_numerator_unit_id,omitzero" validate:"omitempty"`
	// ID of the unit cost's denominator unit (the unit the cost is per).
	UnitCostDenominatorUnitID field.Optional[string] `json:"unit_cost_denominator_unit_id,omitzero" validate:"omitempty"`
}

var sampleUpdatePOLineProductID = apiresource.SampleProductID
var sampleUpdatePOLineQuantityValue = "250"
var sampleUpdatePOLineUnitPriceValue = "15.00"
var sampleUpdatePOLineProductSKU = "RAW-100"
var sampleUpdatePurchaseOrderLineRequest = &UpdatePurchaseOrderLineRequest{
	ProductID:      field.Some(sampleUpdatePOLineProductID),
	ProductSKU:     field.Some(sampleUpdatePOLineProductSKU),
	QuantityValue:  field.Some(sampleUpdatePOLineQuantityValue),
	UnitPriceValue: field.Some(sampleUpdatePOLineUnitPriceValue),
}

func (*UpdatePurchaseOrderLineRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdatePurchaseOrderLineRequest)
}

// Partially updates a purchase order line item.
//
// If the order has already been issued, the receiving order is updated to reflect the remaining quantity to receive.
type UpdatePurchaseOrderLineEndpoint struct{}

func (e *UpdatePurchaseOrderLineEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdatePurchaseOrderLineRequest, *apiresource.PurchaseOrderLine] {
	return (&apiendpoint.APIEndpoint[*UpdatePurchaseOrderLineRequest, *apiresource.PurchaseOrderLine]{
		Title:             "Update Purchase Order Line",
		Method:            http.MethodPatch,
		ContentType:       "application/json",
		Route:             "/v1/operations/purchase-orders/{id}/lines/{line_id}",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdatePurchaseOrderLineRequest) (*apiresource.PurchaseOrderLine, *apierror.APIError) {
			return svc.(PurchaseOrderSvc).UpdatePurchaseOrderLine
		},
	})
}
