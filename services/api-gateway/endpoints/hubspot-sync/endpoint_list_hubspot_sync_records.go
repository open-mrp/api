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

// Request to list the account's HubSpot sync records.
type ListHubspotSyncRecordsRequest struct {
	// Restrict the results to records of this Augno type.
	AugnoType constants.HubspotSyncRecordAugnoType `query:"augno_type" default:"customer" validate:"enum"`
	// Opaque cursor token identifying where the page of results starts.
	//
	// Use the `cursor` value embedded in a previous response's `next_page_url` to fetch the next page. Omit to start from the first page.
	Cursor *string `query:"cursor"`
	// Maximum number of results to return in a single page.
	Limit int32 `query:"limit" default:"100" validate:"min=1,max=1000"`
}

// Lists what the HubSpot sync has written — each Augno record and the HubSpot object it maps to.
//
// Use this to see which customers reached HubSpot, when each was last pushed, and why any of them failed. Results are ordered by Augno record id.
type ListHubspotSyncRecordsEndpoint struct{}

func (e *ListHubspotSyncRecordsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListHubspotSyncRecordsRequest, *apiresource.List[apiresource.HubspotSyncRecord]] {
	return (&apiendpoint.APIEndpoint[*ListHubspotSyncRecordsRequest, *apiresource.List[apiresource.HubspotSyncRecord]]{
		Title:               "List HubSpot Sync Records",
		Method:              http.MethodGet,
		ContentType:         "application/json",
		Route:               "/v1/settings/integrations/hubspot/sync/records",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		Preview:             true,
		ObjectType:          constants.ObjectTypeHubspotSyncRecord,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainIntegrations, Action: types.ActionRead}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListHubspotSyncRecordsRequest) (*apiresource.List[apiresource.HubspotSyncRecord], *apierror.APIError) {
			return svc.(HubspotSyncSvc).ListSyncRecords
		},
	})
}
