package productionstepep

import (
	"context"
	"net/http"
	"time"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list production steps.
type ListProductionStepsRequest struct {
	apiresource.PaginationRequest
	// Filter by produced item IDs.
	ItemIDs []string `query:"item_ids"`
	// Filter by machine IDs.
	MachineIDs []string `query:"machine_ids"`
	// Filter by scanning station IDs.
	ScanningStationIDs []string `query:"scanning_station_ids"`
	// Filter by input step IDs.
	InputStepIDs []string `query:"input_step_ids"`
	// Filter by output step IDs.
	OutputStepIDs []string `query:"output_step_ids"`
	// Filter by start date.
	StartDate *time.Time `query:"start_date"`
	// Filter by end date.
	EndDate *time.Time `query:"end_date"`
}

type ListProductionStepsEndpoint struct{}

func (e *ListProductionStepsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListProductionStepsRequest, *apiresource.List[apiresource.ProductionStep]] {
	return &apiendpoint.APIEndpoint[*ListProductionStepsRequest, *apiresource.List[apiresource.ProductionStep]]{
		Title:             "List Production Steps",
		Description:       "Returns a paginated list of production steps for the current account.",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/production-steps",
		Request:           &ListProductionStepsRequest{},
		Response:          &apiresource.List[apiresource.ProductionStep]{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListProductionStepsRequest) (*apiresource.List[apiresource.ProductionStep], *apierror.APIError) {
			return svc.(ProductionStepSvc).ListProductionSteps
		},
	}
}
