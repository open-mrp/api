package salesorderep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to update a sales order line.
type UpdateSalesOrderLineRequest struct {
	// Sales order ID.
	SalesOrderID string `path:"id" validate:"required"`
	// Sales order line ID.
	SalesOrderLineID string `path:"line_id" validate:"required"`
	// Product ID.
	ProductID *string `json:"product_id,omitempty" nullable:"false" validate:"omitempty"`
	// Item ID.
	ItemID *string `json:"item_id,omitempty" nullable:"false" validate:"omitempty"`
	// Product SKU.
	ProductSKU *string `json:"product_sku,omitempty" nullable:"false" validate:"omitempty,max=255"`
	// Product description.
	ProductDescription *string `json:"product_description,omitempty" nullable:"false"`
	// Quantity value.
	QuantityValue *string `json:"quantity_value,omitempty" nullable:"false" format:"decimal"`
	// Quantity unit ID.
	QuantityUnitID *string `json:"quantity_unit_id,omitempty" nullable:"false" validate:"omitempty"`
	// Unit price value.
	UnitPriceValue *string `json:"unit_price_value,omitempty" nullable:"false" format:"decimal"`
	// Unit price numerator unit ID.
	UnitPriceNumeratorUnitID *string `json:"unit_price_numerator_unit_id,omitempty" nullable:"false" validate:"omitempty"`
	// Unit price denominator unit ID.
	UnitPriceDenominatorUnitID *string `json:"unit_price_denominator_unit_id,omitempty" nullable:"false" validate:"omitempty"`
	// Unit cost value.
	UnitCostValue *string `json:"unit_cost_value,omitempty" nullable:"false" format:"decimal"`
	// Unit cost numerator unit ID.
	UnitCostNumeratorUnitID *string `json:"unit_cost_numerator_unit_id,omitempty" nullable:"false" validate:"omitempty"`
	// Unit cost denominator unit ID.
	UnitCostDenominatorUnitID *string `json:"unit_cost_denominator_unit_id,omitempty" nullable:"false" validate:"omitempty"`
	// EDI line item ID.
	EdiLineItemID *string `json:"edi_line_item_id,omitempty" nullable:"false" validate:"omitempty"`
}

var sampleUpdateSOLineProductID = apiresource.SampleProductID
var sampleUpdateSOLineQuantityValue = "20"
var sampleUpdateSOLineUnitPriceValue = "30.00"
var sampleUpdateSOLineProductSKU = "WIDGET-001"
var sampleUpdateSalesOrderLineRequest = &UpdateSalesOrderLineRequest{
	ProductID:      &sampleUpdateSOLineProductID,
	ProductSKU:     &sampleUpdateSOLineProductSKU,
	QuantityValue:  &sampleUpdateSOLineQuantityValue,
	UnitPriceValue: &sampleUpdateSOLineUnitPriceValue,
}

func (*UpdateSalesOrderLineRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateSalesOrderLineRequest)
}

// Partially updates a sales order line item.
type UpdateSalesOrderLineEndpoint struct{}

func (e *UpdateSalesOrderLineEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateSalesOrderLineRequest, *apiresource.SalesOrderLineDetail] {
	return (&apiendpoint.APIEndpoint[*UpdateSalesOrderLineRequest, *apiresource.SalesOrderLineDetail]{
		Title:             "Update Sales Order Line",
		Method:            http.MethodPatch,
		ContentType:       "application/json",
		Route:             "/v1/sales/sales-orders/{id}/lines/{line_id}",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateSalesOrderLineRequest) (*apiresource.SalesOrderLineDetail, *apierror.APIError) {
			return svc.(SalesOrderSvc).UpdateSalesOrderLine
		},
	})
}
