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

// CreateSalesOrderLineRequest is the request to create a new line on a sales order.
type CreateSalesOrderLineRequest struct {
	// The ID of the sales order.
	SalesOrderID string `path:"id" validate:"required"`
	apirequest.OrderLineInput
	// The EDI line item ID.
	EdiLineItemID *string `json:"edi_line_item_id,omitempty" validate:"omitempty,max=191"`
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

type CreateSalesOrderLineEndpoint struct{}

func (e *CreateSalesOrderLineEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateSalesOrderLineRequest, *apiresource.SalesOrderLineDetail] {
	return &apiendpoint.APIEndpoint[*CreateSalesOrderLineRequest, *apiresource.SalesOrderLineDetail]{
		Title:             "Create Sales Order Line",
		Description:       "Creates a new line item on a sales order.",
		Method:            http.MethodPost,
		Route:             "/v1/sales/sales-orders/{id}/lines",
		Request:           &CreateSalesOrderLineRequest{},
		Response:          &apiresource.SalesOrderLineDetail{},
		SuccessStatusCode: http.StatusCreated,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateSalesOrderLineRequest) (*apiresource.SalesOrderLineDetail, *apierror.APIError) {
			return svc.(SalesOrderSvc).CreateSalesOrderLine
		},
	}
}
