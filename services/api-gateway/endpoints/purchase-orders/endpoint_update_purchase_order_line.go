package purchaseorderep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to update a purchase order line.
type UpdatePurchaseOrderLineRequest struct {
	// Purchase order ID.
	PurchaseOrderID string `path:"id" validate:"required"`
	// Purchase order line ID.
	PurchaseOrderLineID string `path:"line_id" validate:"required"`
	// Product ID.
	ProductID *string `json:"product_id,omitempty" validate:"omitempty"`
	// Item ID.
	ItemID *string `json:"item_id,omitempty" validate:"omitempty"`
	// Product SKU.
	ProductSKU *string `json:"product_sku,omitempty" validate:"omitempty,max=255"`
	// Product description.
	ProductDescription *string `json:"product_description,omitempty"`
	// Quantity value.
	QuantityValue *string `json:"quantity_value,omitempty" format:"decimal"`
	// Quantity unit ID.
	QuantityUnitID *string `json:"quantity_unit_id,omitempty" validate:"omitempty"`
	// Unit price value.
	UnitPriceValue *string `json:"unit_price_value,omitempty" format:"decimal"`
	// Unit price numerator unit ID.
	UnitPriceNumeratorUnitID *string `json:"unit_price_numerator_unit_id,omitempty" validate:"omitempty"`
	// Unit price denominator unit ID.
	UnitPriceDenominatorUnitID *string `json:"unit_price_denominator_unit_id,omitempty" validate:"omitempty"`
	// Unit cost value.
	UnitCostValue *string `json:"unit_cost_value,omitempty" format:"decimal"`
	// Unit cost numerator unit ID.
	UnitCostNumeratorUnitID *string `json:"unit_cost_numerator_unit_id,omitempty" validate:"omitempty"`
	// Unit cost denominator unit ID.
	UnitCostDenominatorUnitID *string `json:"unit_cost_denominator_unit_id,omitempty" validate:"omitempty"`
}

var sampleUpdatePOLineProductID = apiresource.SampleProductID
var sampleUpdatePOLineQuantityValue = "250"
var sampleUpdatePOLineUnitPriceValue = "15.00"
var sampleUpdatePOLineProductSKU = "RAW-100"
var sampleUpdatePurchaseOrderLineRequest = &UpdatePurchaseOrderLineRequest{
	ProductID:      &sampleUpdatePOLineProductID,
	ProductSKU:     &sampleUpdatePOLineProductSKU,
	QuantityValue:  &sampleUpdatePOLineQuantityValue,
	UnitPriceValue: &sampleUpdatePOLineUnitPriceValue,
}

func (*UpdatePurchaseOrderLineRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdatePurchaseOrderLineRequest)
}

// Partially updates a purchase order line item.
type UpdatePurchaseOrderLineEndpoint struct{}

func (e *UpdatePurchaseOrderLineEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdatePurchaseOrderLineRequest, *apiresource.PurchaseOrderLineDetail] {
	return (&apiendpoint.APIEndpoint[*UpdatePurchaseOrderLineRequest, *apiresource.PurchaseOrderLineDetail]{
		Title:             "Update Purchase Order Line",
		Method:            http.MethodPatch,
		ContentType:       "application/json",
		Route:             "/v1/operations/purchase-orders/{id}/lines/{line_id}",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdatePurchaseOrderLineRequest) (*apiresource.PurchaseOrderLineDetail, *apierror.APIError) {
			return svc.(PurchaseOrderSvc).UpdatePurchaseOrderLine
		},
	})
}
