package analyticsep

import (
	"context"
	"net/http"
	"time"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// AnalyzeOeeRequest is the request to analyze Overall Equipment Effectiveness (OEE).
type AnalyzeOeeRequest struct {
	// The start date for the analysis period.
	StartDate time.Time `json:"start_date" validate:"required"`
	// The end date for the analysis period.
	EndDate time.Time `json:"end_date" validate:"required"`
	// Optional department IDs to filter by.
	DepartmentIDs []string `json:"department_ids,omitempty"`
}

type AnalyzeOeeEndpoint struct{}

func (e *AnalyzeOeeEndpoint) Materialize() *apiendpoint.APIEndpoint[*AnalyzeOeeRequest, *apiresource.AnalyzeOeeResponse] {
	return &apiendpoint.APIEndpoint[*AnalyzeOeeRequest, *apiresource.AnalyzeOeeResponse]{
		Title:             "Analyze OEE",
		Description:       "Returns Overall Equipment Effectiveness (OEE) metrics by department, including good units, waste units, and estimated runtime hours.",
		Method:            http.MethodPut,
		Route:             "/v1/core/analytics/oee",
		ContentType:       "application/json",
		Request:           &AnalyzeOeeRequest{},
		Response:          &apiresource.AnalyzeOeeResponse{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *AnalyzeOeeRequest) (*apiresource.AnalyzeOeeResponse, *apierror.APIError) {
			return svc.(AnalyticsSvc).AnalyzeOee
		},
	}
}
