package analyticsep

import (
	"context"
	"net/http"
	"time"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/field"
)

// ListSalesLinesRequest is the request to page through invoiced sale lines.
type ListSalesLinesRequest struct {
	// Start of the window, by invoice date, inclusive. Set both bounds or neither; omit both to list every invoiced line.
	StartDate field.Optional[time.Time] `json:"starts_at,omitzero"`
	// End of the window, by invoice date, inclusive.
	EndDate field.Optional[time.Time] `json:"ends_at,omitzero"`
	SalesReportFilters
	// Opaque cursor from a previous page's `next_page_url` or `previous_page_url`. Omit for the first page.
	Cursor *string `query:"cursor"`
	// Maximum number of lines to return.
	Limit int32 `query:"limit" default:"50" validate:"min=1,max=500"`
}

// Returns invoiced sale lines newest first, each priced with its quantity in the item's base unit, unit and total price, cost and profit.
//
// The same rows `analyze sales` returns, a page at a time. Sales reps see only their own lines, with unit cost zeroed.
type ListSalesLinesEndpoint struct{}

func (e *ListSalesLinesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListSalesLinesRequest, *apiresource.List[apiresource.SalesEntry]] {
	return (&apiendpoint.APIEndpoint[*ListSalesLinesRequest, *apiresource.List[apiresource.SalesEntry]]{
		Title:               "List Sales Lines",
		Method:              http.MethodPut,
		Route:               "/v1/core/analytics/sales-lines",
		ContentType:         "application/json",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		Preview:             true,
		ReadOnly:            true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainInvoices, Action: types.ActionRead}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListSalesLinesRequest) (*apiresource.List[apiresource.SalesEntry], *apierror.APIError) {
			return svc.(AnalyticsSvc).ListSalesLines
		},
	})
}
