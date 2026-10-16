package analyticsep

import (
	"context"
	"net/http"
	"time"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
	"github.com/open-mrp/apikit/field"
)

// ExportSalesLinesRequest filters which invoiced sale lines land in the exported file.
type ExportSalesLinesRequest struct {
	// Start of the window, by invoice date, inclusive. Set both bounds or neither; omit both to export every invoiced line.
	StartDate field.Optional[time.Time] `json:"starts_at,omitzero"`
	// End of the window, by invoice date, inclusive.
	EndDate field.Optional[time.Time] `json:"ends_at,omitzero"`
	SalesReportFilters
}

// Starts an export of the matching invoiced sale lines, oldest first, and returns the job that tracks it.
//
// The file has one row per invoice line with its pricing, customer and ship-to details. Sales reps export only their own sales, and without the unit cost column. An export matching more than 50,000 lines fails with a request to narrow the filters.
type ExportSalesLinesEndpoint struct{}

func (e *ExportSalesLinesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ExportSalesLinesRequest, *apiresource.AsyncJob] {
	return (&apiendpoint.APIEndpoint[*ExportSalesLinesRequest, *apiresource.AsyncJob]{
		Title:             "Export Sales Lines",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/core/analytics/sales-lines/actions/export",
		SuccessStatusCode: http.StatusAccepted,
		Public:            false,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypeAsyncJob,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainInvoices, Action: types.ActionRead}}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ExportSalesLinesRequest) (*apiresource.AsyncJob, *apierror.APIError) {
			return svc.(AnalyticsSvc).ExportSalesLines
		},
		LocationFunc: func(resp *apiresource.AsyncJob) string {
			return "/v1/core/async-jobs/" + resp.ID
		},
	})
}
