package analyticsep

import (
	"context"
	"net/http"
	"time"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// AnalyzeNewCustomersRequest is the request to analyze new customer acquisition.
type AnalyzeNewCustomersRequest struct {
	// The start date for the analysis period.
	StartDate time.Time `json:"start_date" validate:"required"`
	// The end date for the analysis period.
	EndDate time.Time `json:"end_date" validate:"required"`
	// Optional customer group IDs to filter by.
	CustomerGroupIDs []string `json:"customer_group_ids,omitempty"`
	// Optional sales rep IDs to filter by.
	SalesRepIDs []string `json:"sales_rep_ids,omitempty"`
}

// Returns time series data of new customer acquisitions over a specified date range.
type AnalyzeNewCustomersEndpoint struct{}

func (e *AnalyzeNewCustomersEndpoint) Materialize() *apiendpoint.APIEndpoint[*AnalyzeNewCustomersRequest, *apiresource.AnalyzeNewCustomersResponse] {
	return (&apiendpoint.APIEndpoint[*AnalyzeNewCustomersRequest, *apiresource.AnalyzeNewCustomersResponse]{
		Title:             "Analyze New Customers",
		Method:            http.MethodPut,
		Route:             "/v1/core/analytics/new-customers",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *AnalyzeNewCustomersRequest) (*apiresource.AnalyzeNewCustomersResponse, *apierror.APIError) {
			return svc.(AnalyticsSvc).AnalyzeNewCustomers
		},
	})
}
