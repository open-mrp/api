package hubspotsyncep

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

// Request to cancel an in-progress HubSpot sync job.
type CancelHubspotSyncRequest struct {
	// HubSpot sync job ID.
	JobID string `path:"id" validate:"required"`
}

// Cancels a HubSpot sync job that is still in progress, releasing the account to start a new one.
//
// Use this when a sync is stuck — for example when the worker running it stopped without recording an outcome. The job is marked failed, with the cancelling user recorded in `last_error`. Anything already written to HubSpot stays there; cancelling only stops the run. A sync that has already completed or failed cannot be cancelled.
type CancelHubspotSyncEndpoint struct{}

func (e *CancelHubspotSyncEndpoint) Materialize() *apiendpoint.APIEndpoint[*CancelHubspotSyncRequest, *apiresource.HubspotSyncJob] {
	return (&apiendpoint.APIEndpoint[*CancelHubspotSyncRequest, *apiresource.HubspotSyncJob]{
		Title:             "Cancel HubSpot Sync",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/settings/integrations/hubspot/sync/{id}/actions/cancel",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		AgentTool:         true,
		Preview:           true,
		ObjectType:        constants.ObjectTypeHubspotSyncJob,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainIntegrations, Action: types.ActionUpdate}}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *CancelHubspotSyncRequest) (*apiresource.HubspotSyncJob, *apierror.APIError) {
			return svc.(HubspotSyncSvc).CancelSync
		},
	})
}
