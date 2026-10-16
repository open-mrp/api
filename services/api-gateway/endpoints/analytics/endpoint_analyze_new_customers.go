package analyticsep

import (
	"context"
	"net/http"
	"time"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

// AnalyzeNewCustomersRequest is the request to analyze new customer acquisition.
type AnalyzeNewCustomersRequest struct {
	// The start date for the analysis period.
	StartDate time.Time `json:"starts_at" validate:"required"`
	// The end date for the analysis period.
	EndDate time.Time `json:"ends_at" validate:"required"`
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
		Method:            http.MethodPost,
		Route:             "/v1/core/analytics/new-customers/actions/query",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		ReadOnly:          true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainCustomers, Action: types.ActionRead}}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *AnalyzeNewCustomersRequest) (*apiresource.AnalyzeNewCustomersResponse, *apierror.APIError) {
			return svc.(AnalyticsSvc).AnalyzeNewCustomers
		},
	})
}
