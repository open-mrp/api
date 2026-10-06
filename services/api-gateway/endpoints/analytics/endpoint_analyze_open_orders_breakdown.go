package analyticsep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// AnalyzeOpenOrdersBreakdownRequest selects the open sales orders to total by product.
type AnalyzeOpenOrdersBreakdownRequest struct {
	OpenOrderFilters
	// Opaque cursor from a previous page's `next_page_url` or `previous_page_url`. Omit for the first page.
	Cursor *string `query:"cursor"`
	// Maximum number of products to return.
	Limit int32 `query:"limit" default:"10" validate:"min=1,max=100"`
}

// Returns the quantities on open sales orders by product, most back-ordered first.
//
// Each product's ordered, back-ordered and invoiced quantities are totalled across the open orders' sale lines in the item's base unit, converting each line from its own unit. A product with nothing ordered is left out. Products with the same back-ordered quantity are ordered by item ID. Sales reps see only their own orders.
type AnalyzeOpenOrdersBreakdownEndpoint struct{}

func (e *AnalyzeOpenOrdersBreakdownEndpoint) Materialize() *apiendpoint.APIEndpoint[*AnalyzeOpenOrdersBreakdownRequest, *apiresource.List[apiresource.OpenOrderProduct]] {
	return (&apiendpoint.APIEndpoint[*AnalyzeOpenOrdersBreakdownRequest, *apiresource.List[apiresource.OpenOrderProduct]]{
		Title:               "Analyze Open Orders Breakdown",
		Method:              http.MethodPut,
		Route:               "/v1/core/analytics/open-orders/breakdown",
		ContentType:         "application/json",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		Preview:             true,
		ReadOnly:            true,
		ObjectType:          constants.ObjectTypeOpenOrderProduct,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainInvoices, Action: types.ActionRead}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *AnalyzeOpenOrdersBreakdownRequest) (*apiresource.List[apiresource.OpenOrderProduct], *apierror.APIError) {
			return svc.(AnalyticsSvc).AnalyzeOpenOrdersBreakdown
		},
	})
}
