package analyticsep

import (
	"context"
	"net/http"
	"time"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// AnalyzeSalesRequest is the request to analyze sales data over a date range.
type AnalyzeSalesRequest struct {
	// The start date for the analysis period.
	StartDate time.Time `json:"start_date" validate:"required"`
	// The end date for the analysis period.
	EndDate time.Time `json:"end_date" validate:"required"`
	// Optional product line IDs to filter by.
	ProductLineIDs []string `json:"product_line_ids,omitempty"`
	// Optional customer IDs to filter by.
	CustomerIDs []string `json:"customer_ids,omitempty"`
	// Optional sales rep IDs to filter by.
	SalesRepIDs []string `json:"sales_rep_ids,omitempty"`
	// Optional customer group IDs to filter by.
	CustomerGroupIDs []string `json:"customer_group_ids,omitempty"`
	// Optional search query.
	Query *string `json:"query,omitempty"`
}

// Returns detailed sales entry records over a specified date range.
type AnalyzeSalesEndpoint struct{}

func (e *AnalyzeSalesEndpoint) Materialize() *apiendpoint.APIEndpoint[*AnalyzeSalesRequest, *apiresource.AnalyzeSalesResponse] {
	return (&apiendpoint.APIEndpoint[*AnalyzeSalesRequest, *apiresource.AnalyzeSalesResponse]{
		Title:             "Analyze Sales",
		Method:            http.MethodPut,
		Route:             "/v1/core/analytics/sales",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *AnalyzeSalesRequest) (*apiresource.AnalyzeSalesResponse, *apierror.APIError) {
			return svc.(AnalyticsSvc).AnalyzeSales
		},
	})
}
