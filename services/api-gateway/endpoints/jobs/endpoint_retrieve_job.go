package jobep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

// Request to get a job by ID.
type RetrieveJobRequest struct {
	// Job ID.
	JobID string `path:"id" validate:"required"`
}

// Returns an async job by ID — poll the job named in a `202 Accepted` response's `Location` to observe its outcome.
// A completed export carries the link to its file on `export.url`.
//
// A customer or supplier portal reads only the jobs it started itself; any other job is reported as not found.
type RetrieveJobEndpoint struct{}

func (e *RetrieveJobEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveJobRequest, *apiresource.AsyncJob] {
	return (&apiendpoint.APIEndpoint[*RetrieveJobRequest, *apiresource.AsyncJob]{
		Title:             "Retrieve Async Job",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/core/async-jobs/{id}",
		SuccessStatusCode: http.StatusOK,
		// Public: the polling companion for every async operation, including the public bulk endpoints whose 202 Location points here.
		Public:     true,
		AgentTool:  true,
		ObjectType: constants.ObjectTypeAsyncJob,
		Auth:       authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainJobs, Action: types.ActionRead}}, Counterparty: authz.Counterparties(types.ActionRead)},
		// A completed export's link is a bearer credential, so nothing may hold on to it.
		CacheControl: "no-store",
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveJobRequest) (*apiresource.AsyncJob, *apierror.APIError) {
			return svc.(JobSvc).GetJob
		},
	})
}
