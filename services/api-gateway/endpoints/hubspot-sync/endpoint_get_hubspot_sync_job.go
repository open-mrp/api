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

// Request to retrieve a HubSpot sync job.
type GetHubspotSyncJobRequest struct {
	// HubSpot sync job ID.
	JobID string `path:"id" validate:"required"`
}

// Retrieves a HubSpot sync job, including its current status and the dry-run report produced by the preview pass.
//
// Poll this endpoint to track the progress of a running sync.
type GetHubspotSyncJobEndpoint struct{}

func (e *GetHubspotSyncJobEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetHubspotSyncJobRequest, *apiresource.HubspotSyncJob] {
	return (&apiendpoint.APIEndpoint[*GetHubspotSyncJobRequest, *apiresource.HubspotSyncJob]{
		Title:               "Get HubSpot Sync Job",
		Method:              http.MethodGet,
		ContentType:         "application/json",
		Route:               "/v1/settings/integrations/hubspot/sync/{id}",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		Preview:             true,
		ObjectType:          constants.ObjectTypeHubspotSyncJob,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainIntegrations, Action: types.ActionRead}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetHubspotSyncJobRequest) (*apiresource.HubspotSyncJob, *apierror.APIError) {
			return svc.(HubspotSyncSvc).GetSyncJob
		},
	})
}
