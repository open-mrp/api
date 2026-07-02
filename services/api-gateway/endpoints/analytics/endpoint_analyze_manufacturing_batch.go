package analyticsep

import (
	"context"
	"net/http"
	"time"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	apierror "github.com/augno/api/shared/errors"
)

// AnalyzeManufacturingBatchRequest is the request to analyze manufacturing metrics with a comparison period.
type AnalyzeManufacturingBatchRequest struct {
	// The start date for the current analysis period.
	StartDate time.Time `json:"start_date" validate:"required"`
	// The end date for the current analysis period.
	EndDate time.Time `json:"end_date" validate:"required"`
	// The start date for the comparison period.
	ComparisonStartDate time.Time `json:"comparison_start_date" validate:"required"`
	// The end date for the comparison period.
	ComparisonEndDate time.Time `json:"comparison_end_date" validate:"required"`
	// Optional customer IDs to filter by.
	CustomerIDs []string `json:"customer_ids,omitempty"`
	// Optional product line IDs to filter by.
	ProductLineIDs []string `json:"product_line_ids,omitempty"`
	// Optional customer group IDs to filter by.
	CustomerGroupIDs []string `json:"customer_group_ids,omitempty"`
	// Optional item IDs to filter by.
	ItemIDs []string `json:"item_ids,omitempty"`
}

// Returns manufacturing metrics for a current period compared against a comparison period, including production, costs per unit, margin, quality, and labor efficiency.
type AnalyzeManufacturingBatchEndpoint struct{}

func (e *AnalyzeManufacturingBatchEndpoint) Materialize() *apiendpoint.APIEndpoint[*AnalyzeManufacturingBatchRequest, *apiresource.AnalyzeManufacturingBatchResponse] {
	return (&apiendpoint.APIEndpoint[*AnalyzeManufacturingBatchRequest, *apiresource.AnalyzeManufacturingBatchResponse]{
		Title:               "Analyze Manufacturing Batch",
		Method:              http.MethodPut,
		Route:               "/v1/core/analytics/manufacturing-batch",
		ContentType:         "application/json",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		Preview:             true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainInvoices, Action: types.ActionRead}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *AnalyzeManufacturingBatchRequest) (*apiresource.AnalyzeManufacturingBatchResponse, *apierror.APIError) {
			return svc.(AnalyticsSvc).AnalyzeManufacturingBatch
		},
	})
}
