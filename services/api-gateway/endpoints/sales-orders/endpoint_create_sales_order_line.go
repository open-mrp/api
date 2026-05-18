package salesorderep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apirequest "github.com/augno/api/services/api-gateway/pkg/request"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to create a line on a sales order.
type CreateSalesOrderLineRequest struct {
	// Sales order ID.
	SalesOrderID string `path:"id" validate:"required"`
	apirequest.OrderLineInput
	// EDI line item ID.
	EdiLineItemID *string `json:"edi_line_item_id,omitempty" validate:"omitempty"`
}

var sampleCreateSOLineItemID = apiresource.SampleItemID
var sampleCreateSalesOrderLineRequest = &CreateSalesOrderLineRequest{
	OrderLineInput: apirequest.OrderLineInput{
		ProductID:                  apiresource.SampleProductID,
		ItemID:                     &sampleCreateSOLineItemID,
		ProductSKU:                 "WIDGET-001",
		QuantityValue:              "10",
		QuantityUnitID:             apiresource.SampleUnitID,
		UnitPriceValue:             "25.00",
		UnitPriceNumeratorUnitID:   apiresource.SampleUnitID,
		UnitPriceDenominatorUnitID: apiresource.SampleUnitID,
	},
}

func (*CreateSalesOrderLineRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateSalesOrderLineRequest)
}

// Creates a line item on a sales order.
type CreateSalesOrderLineEndpoint struct{}

func (e *CreateSalesOrderLineEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateSalesOrderLineRequest, *apiresource.SalesOrderLineDetail] {
	return (&apiendpoint.APIEndpoint[*CreateSalesOrderLineRequest, *apiresource.SalesOrderLineDetail]{
		Title:             "Create Sales Order Line",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/sales/sales-orders/{id}/lines",
		SuccessStatusCode: http.StatusCreated,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateSalesOrderLineRequest) (*apiresource.SalesOrderLineDetail, *apierror.APIError) {
			return svc.(SalesOrderSvc).CreateSalesOrderLine
		},
	})
}
