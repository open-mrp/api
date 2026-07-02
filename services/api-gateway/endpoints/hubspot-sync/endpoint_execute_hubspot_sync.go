package hubspotsyncep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to execute a reviewed HubSpot sync job.
type ExecuteHubspotSyncRequest struct {
	// HubSpot sync job ID.
	JobID string `path:"id" validate:"required"`
}

// Executes a reviewed HubSpot sync job, writing companies, contacts, and Closed-Won deals to HubSpot.
//
// Every company review must be resolved first. A failed run can be re-executed to resume where it stopped.
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
