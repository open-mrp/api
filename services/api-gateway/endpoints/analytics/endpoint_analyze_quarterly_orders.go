package analyticsep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// AnalyzeQuarterlyOrdersRequest is the request to analyze quarterly order data.
type AnalyzeQuarterlyOrdersRequest struct {
	// Optional sales rep IDs to filter by.
	SalesRepIDs []string `json:"sales_rep_ids,omitempty"`
	// Optional item IDs to filter by.
	ItemIDs []string `json:"item_ids,omitempty"`
	// Optional product line IDs to filter by.
	ProductLineIDs []string `json:"product_line_ids,omitempty"`
	// Optional customer IDs to filter by.
	CustomerIDs []string `json:"customer_ids,omitempty"`
	// Optional customer group IDs to filter by.
	CustomerGroupIDs []string `json:"customer_group_ids,omitempty"`
}

type AnalyzeQuarterlyOrdersEndpoint struct{}

func (e *AnalyzeQuarterlyOrdersEndpoint) Materialize() *apiendpoint.APIEndpoint[*AnalyzeQuarterlyOrdersRequest, *apiresource.AnalyzeQuarterlyOrdersResponse] {
	return &apiendpoint.APIEndpoint[*AnalyzeQuarterlyOrdersRequest, *apiresource.AnalyzeQuarterlyOrdersResponse]{
		Title:             "Analyze Quarterly Orders",
		Description:       "Returns yearly order totals broken down by quarter.",
		Method:            http.MethodPut,
		Route:             "/v1/core/analytics/quarterly-orders",
		ContentType:       "application/json",
		Request:           &AnalyzeQuarterlyOrdersRequest{},
		Response:          &apiresource.AnalyzeQuarterlyOrdersResponse{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *AnalyzeQuarterlyOrdersRequest) (*apiresource.AnalyzeQuarterlyOrdersResponse, *apierror.APIError) {
			return svc.(AnalyticsSvc).AnalyzeQuarterlyOrders
		},
	}
}
