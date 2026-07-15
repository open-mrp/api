package productionrunep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list production runs.
type ListProductionRunsRequest struct {
	apiresource.PaginationRequest
	// Filter by run status.
	//
	// - `open`: runs that have not been completed.
	// - `closed`: runs that have been completed.
	Status *string `query:"status"`
	// Only return runs containing at least one batch that produces any of these items.
	ItemIDs []string `query:"item_ids"`
	// Only return runs containing at least one batch that used any of these machines.
	MachineIDs []string `query:"machine_ids"`
	// Only return runs created on or after this date (inclusive), formatted as `YYYY-MM-DD`.
	StartDate *string `query:"start_date"`
	// Only return runs created on or before this date (inclusive), formatted as `YYYY-MM-DD`.
	EndDate *string `query:"end_date"`
}

// Returns a paginated list of production runs.
type ListProductionRunsEndpoint struct{}

func (e *ListProductionRunsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListProductionRunsRequest, *apiresource.List[apiresource.ProductionRun]] {
	return (&apiendpoint.APIEndpoint[*ListProductionRunsRequest, *apiresource.List[apiresource.ProductionRun]]{
		Title:             "List Production Runs",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/production-runs",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeProductionRun,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainProductionRuns, Action: types.ActionRead},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListProductionRunsRequest) (*apiresource.List[apiresource.ProductionRun], *apierror.APIError) {
			return svc.(ProductionRunSvc).ListProductionRuns
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeProductionRun,
			Fields:     []string{"responsible_user", "responsible_user.user"},
		}),
	})
}
