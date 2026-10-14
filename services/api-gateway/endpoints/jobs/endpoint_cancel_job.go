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

// Request to cancel a job.
type CancelJobRequest struct {
	// Job ID.
	JobID string `path:"id" validate:"required"`
}

// Cancels an async job and returns it carrying its `canceled` status.
// Work in flight is not interrupted but can no longer settle, and a finished job cannot be cancelled.
type CancelJobEndpoint struct{}

func (e *CancelJobEndpoint) Materialize() *apiendpoint.APIEndpoint[*CancelJobRequest, *apiresource.AsyncJob] {
	return (&apiendpoint.APIEndpoint[*CancelJobRequest, *apiresource.AsyncJob]{
		Title:             "Cancel Async Job",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/core/async-jobs/{id}/actions/cancel",
		SuccessStatusCode: http.StatusOK,
		// Public, alongside the retrieve endpoint: a consumer that starts and polls a public async operation can also cancel it.
		Public:     true,
		AgentTool:  true,
		Preview:    true,
		ObjectType: constants.ObjectTypeAsyncJob,
		// What CancelJob enforces downstream. Undeclared, the coarse gate let every authenticated caller through to a 403 raised in the service.
		Auth: authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainJobs, Action: types.ActionDelete}}},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAsyncJob,
			Fields:     []string{"created_by", "created_by.role"},
		}),
		ServiceHandler: func(svc any) func(ctx context.Context, req *CancelJobRequest) (*apiresource.AsyncJob, *apierror.APIError) {
			return svc.(JobSvc).CancelJob
		},
	})
}
