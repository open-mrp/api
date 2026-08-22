package salesorderep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to reorder a sales order's lines.
type ReorderSalesOrderLinesRequest struct {
	// Sales order ID.
	SalesOrderID string `path:"id" validate:"required"`
	// The order's product-line IDs in the desired display order.
	//
	// Every product line on the order must be listed exactly once. The automatically generated discount and freight lines are kept at the bottom of the list and must not be included.
	LineIDs []string `json:"line_ids" validate:"required,min=1,dive,required"`
}

var sampleReorderSalesOrderLinesRequest = &ReorderSalesOrderLinesRequest{
	LineIDs: []string{apiresource.SampleSalesOrderLineID, apiresource.SampleSalesOrderLineID2},
}

func (*ReorderSalesOrderLinesRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleReorderSalesOrderLinesRequest)
}

// Reorders the product lines on a sales order to match the sequence supplied.
//
// The lines are renumbered from `1` in the given order. Discount and freight lines always stay at the bottom of the list regardless of the sequence given here.
type ReorderSalesOrderLinesEndpoint struct{}

func (e *ReorderSalesOrderLinesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ReorderSalesOrderLinesRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*ReorderSalesOrderLinesRequest, *apiresource.EmptyResource]{
		Title:             "Reorder Sales Order Lines",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/sales/sales-orders/{id}/lines/actions/reorder",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Preview:           true,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainCustomers, Action: types.ActionUpdate},
			{Domain: types.PermissionDomainSuppliers, Action: types.ActionUpdate},
			{Domain: types.PermissionDomainSalesOrders, Action: types.ActionUpdate},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ReorderSalesOrderLinesRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(SalesOrderSvc).ReorderSalesOrderLines
		},
	})
}
