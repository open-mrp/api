package edidclocationep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list DC locations.
type ListDCLocationsRequest struct {
	apiresource.PaginationRequest
}

// Returns a paginated list of DC locations for the current account.
type ListDCLocationsEndpoint struct{}

func (e *ListDCLocationsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListDCLocationsRequest, *apiresource.List[apiresource.DCLocation]] {
	return (&apiendpoint.APIEndpoint[*ListDCLocationsRequest, *apiresource.List[apiresource.DCLocation]]{
		Title:             "List DC Locations",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/dc-locations",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListDCLocationsRequest) (*apiresource.List[apiresource.DCLocation], *apierror.APIError) {
			return svc.(EDIDCLocationSvc).ListDCLocations
		},
	})
}
