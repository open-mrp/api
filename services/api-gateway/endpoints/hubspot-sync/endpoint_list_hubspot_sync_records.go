package hubspotsyncep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/contracts"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/pagination"
)

// Request to list the account's HubSpot sync records.
type ListHubspotSyncRecordsRequest struct {
	// The kind of mapping to list.
	//
	// One request returns one kind of mapping; omit this to list the customer-to-company mappings.
	AugnoType constants.HubspotSyncRecordAugnoType `query:"augno_type" default:"customer"`
	// Opaque cursor token identifying where the page of results starts.
	//
	// Use the `cursor` value embedded in a previous response's `next_page_url` to fetch the next page. Omit to start from the first page.
	Cursor *string `query:"cursor"`
	// Maximum number of results to return in a single page.
	Limit int32 `query:"limit" default:"100" validate:"min=1,max=1000"`
}

var _ contracts.DocumentedType = (*ListHubspotSyncRecordsRequest)(nil)

// SchemaExample documents this endpoint's list query parameters for OpenAPI. The cursor keysets on the record's augno_id, so it is a string cursor rather than the id-based one PaginationRequest documents.
func (*ListHubspotSyncRecordsRequest) SchemaExample() any {
	return map[string]any{
		"augno_type": string(constants.HubspotSyncRecordAugnoTypeCustomer),
		"cursor":     pagination.EncodeDocumentationStringCursor(apiresource.SampleAnalyticsPeriodStart, apiresource.SampleHubspotSyncRecordID),
		"limit":      int64(100),
	}
}

// Lists the mappings the HubSpot sync has recorded for the account — each OpenMRP record and the HubSpot object it maps to.
//
// A mapping is recorded as soon as the sync resolves a record's HubSpot object, which for a confidently matched customer happens during the read-only preview, before anything has been written to HubSpot. Results are ordered by OpenMRP record id.
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
