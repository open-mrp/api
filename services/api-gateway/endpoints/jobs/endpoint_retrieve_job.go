package jobep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to get a job by ID.
type RetrieveJobRequest struct {
	// Job ID.
	JobID string `path:"id" validate:"required"`
}

// Returns a job by ID — poll the job named in a `202 Accepted` response's `Location` to observe its outcome.
// A completed export carries the link to its file on `export.url`.
type RetrieveJobEndpoint struct{}

func (e *RetrieveJobEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveJobRequest, *apiresource.Job] {
	return (&apiendpoint.APIEndpoint[*RetrieveJobRequest, *apiresource.Job]{
		Title:             "Retrieve Job",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/core/jobs/{id}",
		SuccessStatusCode: http.StatusOK,
		// Public: the polling companion for every async operation, including the public bulk endpoints whose 202 Location points here.
		Public:     true,
		AgentTool:  true,
		Preview:    true,
		ObjectType: constants.ObjectTypeJob,
		// The OR-set checkJobReadPermission enforces: jobs:read for an internal actor reading its own account, customers:read / suppliers:read when the target is an external account.
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainJobs, Action: types.ActionRead},
			{Domain: types.PermissionDomainCustomers, Action: types.ActionRead},
			{Domain: types.PermissionDomainSuppliers, Action: types.ActionRead},
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeJob,
			Fields:     []string{"created_by", "created_by.role"},
		}),
		// A completed export's link is a bearer credential, so nothing may hold on to it.
		CacheControl: "no-store",
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveJobRequest) (*apiresource.Job, *apierror.APIError) {
			return svc.(JobSvc).GetJob
		},
	})
}
