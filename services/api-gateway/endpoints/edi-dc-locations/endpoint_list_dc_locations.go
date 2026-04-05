package edidclocationep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// ListDCLocationsRequest is the request to list DC locations with optional search.
type ListDCLocationsRequest struct {
	apiresource.PaginationRequest
}

type ListDCLocationsEndpoint struct{}

func (e *ListDCLocationsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListDCLocationsRequest, *apiresource.List[apiresource.DCLocation]] {
	return &apiendpoint.APIEndpoint[*ListDCLocationsRequest, *apiresource.List[apiresource.DCLocation]]{
		Title:             "List DC Locations",
		Description:       "Returns a paginated list of DC locations for the current account.",
		Method:            http.MethodGet,
		Route:             "/v1/operations/dc-locations",
		Request:           &ListDCLocationsRequest{},
		Response:          &apiresource.List[apiresource.DCLocation]{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListDCLocationsRequest) (*apiresource.List[apiresource.DCLocation], *apierror.APIError) {
			return svc.(EDIDCLocationSvc).ListDCLocations
		},
	}
}
