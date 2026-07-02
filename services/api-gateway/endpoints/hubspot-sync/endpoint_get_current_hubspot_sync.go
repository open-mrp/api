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

// Request to retrieve the account's current HubSpot sync job.
type GetCurrentHubspotSyncRequest struct{}

// Retrieves the account's most recent HubSpot sync job so a dashboard can resume an in-progress sync after a refresh.
//
// Returns a not-found error when no sync has ever been started for the account.
type GetCurrentHubspotSyncEndpoint struct{}

func (e *GetCurrentHubspotSyncEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetCurrentHubspotSyncRequest, *apiresource.HubspotSyncJob] {
	return (&apiendpoint.APIEndpoint[*GetCurrentHubspotSyncRequest, *apiresource.HubspotSyncJob]{
		Title:               "Get Current HubSpot Sync",
		Method:              http.MethodGet,
		ContentType:         "application/json",
		Route:               "/v1/settings/integrations/hubspot/sync/current",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		Preview:             true,
		ObjectType:          constants.ObjectTypeHubspotSyncJob,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainIntegrations, Action: types.ActionRead}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetCurrentHubspotSyncRequest) (*apiresource.HubspotSyncJob, *apierror.APIError) {
			return svc.(HubspotSyncSvc).GetCurrentSyncJob
		},
	})
}
