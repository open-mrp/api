package batchep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	types "github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to list batches for a scanning station.
type ListBatchesByScanningStationRequest struct {
	// Scanning station ID.
	ScanningStationID string `path:"id" validate:"required"`
	apiresource.PaginationRequest
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
		Preview:           true,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainBatches, Action: types.ActionRead},
		},
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
