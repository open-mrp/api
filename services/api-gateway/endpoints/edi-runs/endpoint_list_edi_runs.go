package edirunep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list EDI runs.
type ListEDIRunsRequest struct {
	apiresource.PaginationRequest
	// Filters runs by outcome.
	//
	// Pass `true` to return only successful runs or `false` to return only failed runs. Omit to return runs regardless of outcome.
	HasSucceeded *bool `query:"has_succeeded"`
}

// Returns a paginated list of EDI runs for the target account.
//
// Runs are ordered by completion time, most recent first. The `q` search term matches the EDI run ID.
type ListEDIRunsEndpoint struct{}

func (e *ListEDIRunsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListEDIRunsRequest, *apiresource.List[apiresource.EDIRun]] {
	return (&apiendpoint.APIEndpoint[*ListEDIRunsRequest, *apiresource.List[apiresource.EDIRun]]{
		Title:             "List EDI Runs",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/edi-runs",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainEdiRuns, Action: types.ActionRead},
		},
		ObjectType: constants.ObjectTypeEDIRun,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListEDIRunsRequest) (*apiresource.List[apiresource.EDIRun], *apierror.APIError) {
			return svc.(EDIRunSvc).ListEDIRuns
		},
	})
}
