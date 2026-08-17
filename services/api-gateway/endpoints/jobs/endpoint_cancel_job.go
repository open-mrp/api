package jobep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to cancel a job.
type CancelJobRequest struct {
	// Job ID.
	JobID string `path:"id" validate:"required"`
}

// Cancels a job and returns it carrying its `cancelled` status.
// Work in flight is not interrupted but can no longer settle, and a finished job cannot be cancelled.
type CancelJobEndpoint struct{}

func (e *CancelJobEndpoint) Materialize() *apiendpoint.APIEndpoint[*CancelJobRequest, *apiresource.Job] {
	return (&apiendpoint.APIEndpoint[*CancelJobRequest, *apiresource.Job]{
		Title:             "Cancel Job",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/core/jobs/{id}/cancel",
		SuccessStatusCode: http.StatusOK,
		// Public, alongside the retrieve endpoint: a consumer that starts and polls a public async operation can also cancel it.
		Public:     true,
		Preview:    true,
		ObjectType: constants.ObjectTypeJob,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeJob,
			Fields:     []string{"created_by", "created_by.role"},
		}),
		ServiceHandler: func(svc any) func(ctx context.Context, req *CancelJobRequest) (*apiresource.Job, *apierror.APIError) {
			return svc.(JobSvc).CancelJob
		},
	})
}
