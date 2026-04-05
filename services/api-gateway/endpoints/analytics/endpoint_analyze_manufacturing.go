package analyticsep

import (
	"context"
	"net/http"
	"time"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// AnalyzeManufacturingRequest is the request to analyze a single manufacturing metric.
type AnalyzeManufacturingRequest struct {
	// The start date for the analysis period.
	StartDate time.Time `json:"start_date" validate:"required"`
	// The end date for the analysis period.
	EndDate time.Time `json:"end_date" validate:"required"`
	// The type of manufacturing analytics to compute.
	Type string `json:"type" validate:"required"`
}

type AnalyzeManufacturingEndpoint struct{}

func (e *AnalyzeManufacturingEndpoint) Materialize() *apiendpoint.APIEndpoint[*AnalyzeManufacturingRequest, *apiresource.AnalyzeManufacturingResponse] {
	return &apiendpoint.APIEndpoint[*AnalyzeManufacturingRequest, *apiresource.AnalyzeManufacturingResponse]{
		Title:             "Analyze Manufacturing",
		Description:       "Returns a single manufacturing analytics metric for a specified date range and type.",
		Method:            http.MethodPut,
		Route:             "/v1/core/analytics/manufacturing",
		ContentType:       "application/json",
		Request:           &AnalyzeManufacturingRequest{},
		Response:          &apiresource.AnalyzeManufacturingResponse{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *AnalyzeManufacturingRequest) (*apiresource.AnalyzeManufacturingResponse, *apierror.APIError) {
			return svc.(AnalyticsSvc).AnalyzeManufacturing
		},
	}
}
