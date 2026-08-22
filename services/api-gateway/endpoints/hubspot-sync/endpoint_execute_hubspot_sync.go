package hubspotsyncep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to execute a reviewed HubSpot sync job.
type ExecuteHubspotSyncRequest struct {
	// HubSpot sync job ID.
	JobID string `path:"id" validate:"required"`
}

// Executes a reviewed HubSpot sync job, writing companies, contacts, and Closed-Won deals to HubSpot.
//
// Every company review must be resolved or skipped first, and the job's preview pass must have finished — a sync that failed mid-preview has incomplete matches and cannot be executed, so start a new one instead. Writing happens in the background: this returns as soon as the job is claimed, and the job moves to `executing`. Calling it again while the sync is running is rejected, but a run that failed part-way can be executed again to resume where it stopped.
type ExecuteHubspotSyncEndpoint struct{}

func (e *ExecuteHubspotSyncEndpoint) Materialize() *apiendpoint.APIEndpoint[*ExecuteHubspotSyncRequest, *apiresource.HubspotSyncJob] {
	return (&apiendpoint.APIEndpoint[*ExecuteHubspotSyncRequest, *apiresource.HubspotSyncJob]{
		Title:               "Execute HubSpot Sync",
		Method:              http.MethodPost,
		ContentType:         "application/json",
		Route:               "/v1/settings/integrations/hubspot/sync/{id}/actions/execute",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		Preview:             true,
		ObjectType:          constants.ObjectTypeHubspotSyncJob,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainIntegrations, Action: types.ActionUpdate}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ExecuteHubspotSyncRequest) (*apiresource.HubspotSyncJob, *apierror.APIError) {
			return svc.(HubspotSyncSvc).ExecuteSync
		},
	})
}
