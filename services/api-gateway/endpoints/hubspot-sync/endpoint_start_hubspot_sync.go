package hubspotsyncep

import (
	"context"
	"net/http"
	"time"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/field"
)

// Request to start a HubSpot backfill.
type StartHubspotSyncRequest struct {
	// Orders placed on or after this date are backfilled as Closed-Won deals during the sync.
	//
	// Only the UTC date is used, so the whole of that day is included regardless of the time of day given. Omit to sync companies and contacts only, with no historical deals.
	GoLiveCutoffAt field.Optional[time.Time] `json:"go_live_cutoff_at,omitzero"`
}

var sampleStartHubspotSyncRequest = &StartHubspotSyncRequest{
	GoLiveCutoffAt: field.Some(apiresource.SampleAnalyticsPeriodStart),
}

func (*StartHubspotSyncRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleStartHubspotSyncRequest)
}

// Starts a one-time HubSpot backfill for the account and kicks off the read-only preview pass.
//
// The job matches existing customers to HubSpot companies and produces a dry-run report; it writes nothing to HubSpot until the review queue is resolved and the sync is executed. Poll the returned job to know when the preview has finished.
//
// Only one sync can be underway at a time: starting another while a sync is previewing, awaiting review, or executing is rejected. If the account has no active HubSpot integration, the job is still created but its preview pass fails immediately.
type StartHubspotSyncEndpoint struct{}

func (e *StartHubspotSyncEndpoint) Materialize() *apiendpoint.APIEndpoint[*StartHubspotSyncRequest, *apiresource.HubspotSyncJob] {
	return (&apiendpoint.APIEndpoint[*StartHubspotSyncRequest, *apiresource.HubspotSyncJob]{
		Title:               "Start HubSpot Sync",
		Method:              http.MethodPost,
		ContentType:         "application/json",
		Route:               "/v1/settings/integrations/hubspot/sync",
		SuccessStatusCode:   http.StatusCreated,
		Public:              false,
		Preview:             true,
		ObjectType:          constants.ObjectTypeHubspotSyncJob,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainIntegrations, Action: types.ActionUpdate}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *StartHubspotSyncRequest) (*apiresource.HubspotSyncJob, *apierror.APIError) {
			return svc.(HubspotSyncSvc).StartSync
		},
		LocationFunc: func(resp *apiresource.HubspotSyncJob) string {
			return "/v1/settings/integrations/hubspot/sync/" + resp.ID
		},
	})
}
