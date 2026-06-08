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
	// Item ID.
	ItemID field.Optional[string] `json:"item_id,omitzero" validate:"omitempty"`
	// Product SKU.
	ProductSKU field.Optional[string] `json:"product_sku,omitzero" validate:"omitempty,max=255"`
	// Product description.
	ProductDescription field.Optional[string] `json:"product_description,omitzero"`
	// Quantity value.
	QuantityValue field.Optional[string] `json:"quantity_value,omitzero" format:"decimal"`
	// Quantity unit ID.
	QuantityUnitID field.Optional[string] `json:"quantity_unit_id,omitzero" validate:"omitempty"`
	// Unit price value.
	UnitPriceValue field.Optional[string] `json:"unit_price_value,omitzero" format:"decimal"`
	// Unit price numerator unit ID.
	UnitPriceNumeratorUnitID field.Optional[string] `json:"unit_price_numerator_unit_id,omitzero" validate:"omitempty"`
	// Unit price denominator unit ID.
	UnitPriceDenominatorUnitID field.Optional[string] `json:"unit_price_denominator_unit_id,omitzero" validate:"omitempty"`
	// Unit cost value.
	UnitCostValue field.Optional[string] `json:"unit_cost_value,omitzero" format:"decimal"`
	// Unit cost numerator unit ID.
	UnitCostNumeratorUnitID field.Optional[string] `json:"unit_cost_numerator_unit_id,omitzero" validate:"omitempty"`
	// Unit cost denominator unit ID.
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
