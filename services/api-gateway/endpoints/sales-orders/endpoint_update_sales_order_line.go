package salesorderep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// UpdateSalesOrderLineRequest is the request to update a sales order line.
type UpdateSalesOrderLineRequest struct {
	// The ID of the sales order.
	SalesOrderID string `path:"id" validate:"required"`
	// The ID of the sales order line.
	SalesOrderLineID string `path:"lineId" validate:"required"`
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
	// The EDI line item ID.
	EdiLineItemID *string `json:"edi_line_item_id,omitempty" nullable:"true"`
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

type UpdateSalesOrderLineEndpoint struct{}

func (e *UpdateSalesOrderLineEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateSalesOrderLineRequest, *apiresource.SalesOrderLineDetail] {
	return &apiendpoint.APIEndpoint[*UpdateSalesOrderLineRequest, *apiresource.SalesOrderLineDetail]{
		Title:             "Update Sales Order Line",
		Description:       "Partially updates a sales order line item.",
		Method:            http.MethodPatch,
		Route:             "/v1/sales/sales-orders/{id}/lines/{lineId}",
		Request:           &UpdateSalesOrderLineRequest{},
		Response:          &apiresource.SalesOrderLineDetail{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateSalesOrderLineRequest) (*apiresource.SalesOrderLineDetail, *apierror.APIError) {
			return svc.(SalesOrderSvc).UpdateSalesOrderLine
		},
	}
}
