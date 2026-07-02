package salesorderep

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

// Request to create a line on a sales order.
type CreateSalesOrderLineRequest struct {
	// Sales order ID.
	SalesOrderID string `path:"id" validate:"required"`
	apirequest.OrderLineInput
}

var sampleCreateSOLineItemID = apiresource.SampleItemID
var sampleCreateSalesOrderLineRequest = &CreateSalesOrderLineRequest{
	OrderLineInput: apirequest.OrderLineInput{
		ProductID:                  apiresource.SampleProductID,
		ItemID:                     field.Some(sampleCreateSOLineItemID),
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

func (e *CreateSalesOrderLineEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateSalesOrderLineRequest, *apiresource.SalesOrderLine] {
	return (&apiendpoint.APIEndpoint[*CreateSalesOrderLineRequest, *apiresource.SalesOrderLine]{
		Title:             "Create Sales Order Line",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/sales/sales-orders/{id}/lines",
		SuccessStatusCode: http.StatusCreated,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeSalesOrderLine,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainCustomers, Action: types.ActionUpdate},
			{Domain: types.PermissionDomainSuppliers, Action: types.ActionUpdate},
			{Domain: types.PermissionDomainSalesOrders, Action: types.ActionUpdate},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateSalesOrderLineRequest) (*apiresource.SalesOrderLine, *apierror.APIError) {
			return svc.(SalesOrderSvc).CreateSalesOrderLine
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeSalesOrderLine,
			Fields:     []string{"product", "quantity_ordered", "unit_price", "unit_cost", "totals"},
		}),
	})
}
