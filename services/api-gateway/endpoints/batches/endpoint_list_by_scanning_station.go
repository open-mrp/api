package batchep

import (
	"context"
	"net/http"

	"github.com/open-mrp/apikit/object"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	types "github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

// Request to list batches for a scanning station.
type ListBatchesByScanningStationRequest struct {
	// Scanning station ID.
	ScanningStationID string `path:"id" validate:"required"`
	object.ListRequest
	// Free-text search term used to filter results.
	//
	// Which fields are matched against the term varies by endpoint.
	Query *string `query:"q" validate:"omitempty,max=500"`
}

// Returns a paginated list of the batches scanned at a given scanning station, most recently scanned first.
//
// Only batches that have actually been scanned at the station appear. Batches created there by a move, merge, or split are attached to the station but never marked as scanned, so they are not listed. The search term matches on item SKU.
type ListBatchesByScanningStationEndpoint struct{}

func (e *ListBatchesByScanningStationEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListBatchesByScanningStationRequest, *apiresource.List[apiresource.Batch]] {
	return (&apiendpoint.APIEndpoint[*ListBatchesByScanningStationRequest, *apiresource.List[apiresource.Batch]]{
		Title:             "List Batches by Scanning Station",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/scanning-stations/{id}/batches",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		AgentTool:         true,
		Preview:           true,
		Auth: authz.Policy{Permissions: []types.Permission{
			{Domain: types.PermissionDomainBatches, Action: types.ActionRead},
		}},
		ObjectType: constants.ObjectTypeBatch,
		// A batch reports three measures; the units they are counted in are records of their own.
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeBatch,
			Fields:     []string{"quantity.unit", "seconds.unit", "waste.unit"},
		}),
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListBatchesByScanningStationRequest) (*apiresource.List[apiresource.Batch], *apierror.APIError) {
			return svc.(BatchSvc).ListBatchesByScanningStation
		},
	})
}
