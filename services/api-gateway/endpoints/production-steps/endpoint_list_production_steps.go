package productionstepep

import (
	"context"
	"net/http"
	"time"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to list production steps.
type ListProductionStepsRequest struct {
	apiresource.PaginationRequest
	// Only return steps that produce or consume any of these items.
	ItemIDs []string `query:"item_ids"`
	// Only return steps with any of these machines assigned.
	MachineIDs []string `query:"machine_ids"`
	// Only return steps assigned to any of these scanning stations.
	ScanningStationIDs []string `query:"scanning_station_ids"`
	// Only return steps that are directly fed by any of these upstream steps.
	InputStepIDs []string `query:"input_step_ids"`
	// Only return steps that feed directly into any of these downstream steps.
	OutputStepIDs []string `query:"output_step_ids"`
	// Only return steps created on or after this timestamp (inclusive).
	StartDate *time.Time `query:"starts_at"`
	// Only return steps created on or before this timestamp (inclusive).
	EndDate *time.Time `query:"ends_at"`
}

// Returns a paginated list of production steps for the current account, newest first.
//
// The `q` search term matches against the step name. Filters combine with AND, while the values within a single filter combine with OR.
type ListProductionStepsEndpoint struct{}

func (e *ListProductionStepsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListProductionStepsRequest, *apiresource.List[apiresource.ProductionStep]] {
	return (&apiendpoint.APIEndpoint[*ListProductionStepsRequest, *apiresource.List[apiresource.ProductionStep]]{
		Title:               "List Production Steps",
		Method:              http.MethodGet,
		ContentType:         "application/json",
		Route:               "/v1/operations/production-steps",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		Preview:             true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainProductionSteps, Action: types.ActionRead}},
		ObjectType:          constants.ObjectTypeProductionStep,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListProductionStepsRequest) (*apiresource.List[apiresource.ProductionStep], *apierror.APIError) {
			return svc.(ProductionStepSvc).ListProductionSteps
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeProductionStep,
			Fields: []string{
				"production",
				"production.produced_item",
				"consumptions",
				"consumptions.consumed_item",
				"consumptions.quantity",
				"consumptions.waste_quantity",
				"machines",
				"scanning_station",
				"department",
				"in_steps",
				"out_steps",
			},
		}),
	})
}
