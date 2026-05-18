package analyticsep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// AnalyzeOrdersRequest is the request to analyze order data.
type AnalyzeOrdersRequest struct {
	// Optional product line IDs to filter by.
	ProductLineIDs []string `json:"product_line_ids,omitempty"`
	// Optional customer IDs to filter by.
	CustomerIDs []string `json:"customer_ids,omitempty"`
	// Optional sales rep IDs to filter by.
	SalesRepIDs []string `json:"sales_rep_ids,omitempty"`
	// Optional customer group IDs to filter by.
	CustomerGroupIDs []string `json:"customer_group_ids,omitempty"`
}

// Returns detailed order entry records.
type AnalyzeOrdersEndpoint struct{}

func (e *AnalyzeOrdersEndpoint) Materialize() *apiendpoint.APIEndpoint[*AnalyzeOrdersRequest, *apiresource.AnalyzeOrdersResponse] {
	return (&apiendpoint.APIEndpoint[*AnalyzeOrdersRequest, *apiresource.AnalyzeOrdersResponse]{
		Title:             "Analyze Orders",
		Method:            http.MethodPut,
		Route:             "/v1/core/analytics/orders",
		ContentType:       "application/json",
		Request:           &AnalyzeOrdersRequest{},
		Response:          &apiresource.AnalyzeOrdersResponse{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *AnalyzeOrdersRequest) (*apiresource.AnalyzeOrdersResponse, *apierror.APIError) {
			return svc.(AnalyticsSvc).AnalyzeOrders
		},
	}).WithDocSource(e)
}
