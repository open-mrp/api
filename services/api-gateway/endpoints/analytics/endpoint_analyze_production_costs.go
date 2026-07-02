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

// AnalyzeProductionCostsRequest is the request to analyze production costs.
type AnalyzeProductionCostsRequest struct {
	// Optional start date for the analysis period.
	StartDate *time.Time `json:"start_date,omitempty"`
	// Optional end date for the analysis period.
	EndDate *time.Time `json:"end_date,omitempty"`
	// Optional item IDs to filter by.
	ItemIDs []string `json:"item_ids,omitempty"`
	// Optional product line IDs to filter by.
	ProductLineIDs []string `json:"product_line_ids,omitempty"`
	// Optional department IDs to filter by.
	DepartmentIDs []string `json:"department_ids,omitempty"`
	// Optional category IDs to filter by.
	CategoryIDs []string `json:"category_ids,omitempty"`
}

// Returns aggregated production cost breakdowns by department and category.
type AnalyzeProductionCostsEndpoint struct{}

func (e *AnalyzeProductionCostsEndpoint) Materialize() *apiendpoint.APIEndpoint[*AnalyzeProductionCostsRequest, *apiresource.AnalyzeProductionCostsResponse] {
	return (&apiendpoint.APIEndpoint[*AnalyzeProductionCostsRequest, *apiresource.AnalyzeProductionCostsResponse]{
		Title:               "Analyze Production Costs",
		Method:              http.MethodPut,
		Route:               "/v1/core/analytics/production-costs",
		ContentType:         "application/json",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		Preview:             true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainBatches, Action: types.ActionRead}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *AnalyzeProductionCostsRequest) (*apiresource.AnalyzeProductionCostsResponse, *apierror.APIError) {
			return svc.(AnalyticsSvc).AnalyzeProductionCosts
		},
	})
}
