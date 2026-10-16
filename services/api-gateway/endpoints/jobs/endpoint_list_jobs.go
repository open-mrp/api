package jobep

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

// Request to list async jobs.
type ListJobsRequest struct {
	object.ListRequest
	// Only return jobs of any of these types: `bulk_create`, `bulk_upsert`, `export`, `pack_pick`, `ship_shipment` or `merge_customers`.
	Types []constants.JobType `query:"types"`
	// Only return jobs with any of these statuses: `queued`, `running`, `completed`, `failed` or `canceled`.
	Statuses []object.AsyncJobStatus `query:"statuses"`
}

// Returns a list of async jobs.
//
// Sorted by creation date, newest first. A customer or supplier portal sees only the jobs it started itself.
type ListJobsEndpoint struct{}

func (e *ListJobsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListJobsRequest, *apiresource.List[apiresource.AsyncJob]] {
	return (&apiendpoint.APIEndpoint[*ListJobsRequest, *apiresource.List[apiresource.AsyncJob]]{
		Title:             "List Async Jobs",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/core/async-jobs",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypeAsyncJob,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainJobs, Action: types.ActionRead}}, Counterparty: authz.Counterparties(types.ActionRead)},
		// A completed export's link is a bearer credential, so nothing may hold on to it.
		CacheControl: "no-store",
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListJobsRequest) (*apiresource.List[apiresource.AsyncJob], *apierror.APIError) {
			return svc.(JobSvc).ListJobs
		},
	})
}
