package purchaseorderep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// UpdatePurchaseOrderLineRequest is the request to update a purchase order line.
type UpdatePurchaseOrderLineRequest struct {
	// The ID of the purchase order.
	PurchaseOrderID string `path:"id" validate:"required"`
	// The ID of the purchase order line.
	PurchaseOrderLineID string `path:"lineId" validate:"required"`
	// The product ID.
	ProductID *string `json:"product_id,omitempty" nullable:"true"`
	// The item ID.
	ItemID *string `json:"item_id,omitempty" nullable:"true"`
	// The product SKU.
	ProductSKU *string `json:"product_sku,omitempty"`
	// The product description.
	ProductDescription *string `json:"product_description,omitempty"`
	// The quantity value.
	QuantityValue *string `json:"quantity_value,omitempty" format:"decimal"`
	// The quantity unit ID.
	QuantityUnitID *string `json:"quantity_unit_id,omitempty" nullable:"true"`
	// The unit price value.
	UnitPriceValue *string `json:"unit_price_value,omitempty" format:"decimal"`
	// The unit price numerator unit ID.
	UnitPriceNumeratorUnitID *string `json:"unit_price_numerator_unit_id,omitempty" nullable:"true"`
	// The unit price denominator unit ID.
	UnitPriceDenominatorUnitID *string `json:"unit_price_denominator_unit_id,omitempty" nullable:"true"`
	// The unit cost value.
	UnitCostValue *string `json:"unit_cost_value,omitempty" format:"decimal"`
	// The unit cost numerator unit ID.
	UnitCostNumeratorUnitID *string `json:"unit_cost_numerator_unit_id,omitempty" nullable:"true"`
	// The unit cost denominator unit ID.
	UnitCostDenominatorUnitID *string `json:"unit_cost_denominator_unit_id,omitempty" nullable:"true"`
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

type UpdatePurchaseOrderLineEndpoint struct{}

func (e *UpdatePurchaseOrderLineEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdatePurchaseOrderLineRequest, *apiresource.PurchaseOrderLineDetail] {
	return &apiendpoint.APIEndpoint[*UpdatePurchaseOrderLineRequest, *apiresource.PurchaseOrderLineDetail]{
		Title:             "Update Purchase Order Line",
		Description:       "Partially updates a purchase order line item.",
		Method:            http.MethodPatch,
		Route:             "/v1/operations/purchase-orders/{id}/lines/{lineId}",
		Request:           &UpdatePurchaseOrderLineRequest{},
		Response:          &apiresource.PurchaseOrderLineDetail{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdatePurchaseOrderLineRequest) (*apiresource.PurchaseOrderLineDetail, *apierror.APIError) {
			return svc.(PurchaseOrderSvc).UpdatePurchaseOrderLine
		},
	}
}
