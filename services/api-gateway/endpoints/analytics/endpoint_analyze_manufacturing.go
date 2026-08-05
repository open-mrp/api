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

// AnalyzeManufacturingRequest is the request to analyze a single manufacturing metric.
type AnalyzeManufacturingRequest struct {
	// The start date for the analysis period.
	StartDate time.Time `json:"starts_at" validate:"required"`
	// The end date for the analysis period.
	EndDate time.Time `json:"ends_at" validate:"required"`
	// The type of manufacturing analytics to compute.
	Type string `json:"type" validate:"required"`
}

// Returns a single manufacturing analytics metric for a specified date range and type.
type AnalyzeManufacturingEndpoint struct{}

func (e *AnalyzeManufacturingEndpoint) Materialize() *apiendpoint.APIEndpoint[*AnalyzeManufacturingRequest, *apiresource.AnalyzeManufacturingResponse] {
	return (&apiendpoint.APIEndpoint[*AnalyzeManufacturingRequest, *apiresource.AnalyzeManufacturingResponse]{
		Title:               "Analyze Manufacturing",
		Method:              http.MethodPut,
		Route:               "/v1/core/analytics/manufacturing",
		ContentType:         "application/json",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		Preview:             true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainInvoices, Action: types.ActionRead}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *AnalyzeManufacturingRequest) (*apiresource.AnalyzeManufacturingResponse, *apierror.APIError) {
			return svc.(AnalyticsSvc).AnalyzeManufacturing
		},
	})
}
