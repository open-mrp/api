package productionrunep

import (
	"context"
	"net/http"

	"github.com/open-mrp/apikit/object"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

// Request to list production runs.
type ListProductionRunsRequest struct {
	object.ListRequest
	// Free-text search term used to filter results.
	//
	// Which fields are matched against the term varies by endpoint.
	Query *string `query:"q" validate:"omitempty,max=500"`
	// Filter by run status.
	//
	// A run is `open` until every batch in it has been scanned or deleted, at which point it completes and becomes `closed`. Only open runs are returned when this filter is omitted; ask for `closed` to see finished runs, or `all` for both.
	Status *constants.ProductionRunStatus `query:"status"`
	// Only return runs containing at least one batch that produces any of these items.
	ItemIDs []string `query:"item_ids"`
	// Only return runs containing at least one batch that used any of these machines.
	MachineIDs []string `query:"machine_ids"`
	// Only return runs created at or after this time: a `YYYY-MM-DD` date (the start of that day in UTC) or an RFC 3339 timestamp.
	StartDate *string `query:"starts_at" validate:"omitempty,date_filter"`
	// Only return runs created before this time: a `YYYY-MM-DD` date (the start of that day in UTC) or an RFC 3339 timestamp.
	//
	// A date cuts off at the start of the given day, so runs created during that day are not returned; pass the following day to include them.
	EndDate *string `query:"ends_at" validate:"omitempty,date_filter"`
}

// Returns a paginated list of production runs, most recently created first.
//
// The `q` search term matches the run number.
type ListProductionRunsEndpoint struct{}

func (e *ListProductionRunsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListProductionRunsRequest, *apiresource.List[apiresource.ProductionRun]] {
	return (&apiendpoint.APIEndpoint[*ListProductionRunsRequest, *apiresource.List[apiresource.ProductionRun]]{
		Title:             "List Production Runs",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/production-runs",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		AgentTool:         true,
		Preview:           true,
		ObjectType:        constants.ObjectTypeProductionRun,
		Auth: authz.Policy{Permissions: []types.Permission{
			{Domain: types.PermissionDomainProductionRuns, Action: types.ActionRead},
		}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListProductionRunsRequest) (*apiresource.List[apiresource.ProductionRun], *apierror.APIError) {
			return svc.(ProductionRunSvc).ListProductionRuns
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeProductionRun,
			Fields:     []string{"responsible_user", "responsible_user.user"},
		}),
	})
}
