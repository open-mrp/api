package locationep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// ListLocationTypesRequest is the request to list location types.
type ListLocationTypesRequest struct {
	apiresource.PaginationRequest
}

type ListLocationTypesEndpoint struct{}

func (e *ListLocationTypesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListLocationTypesRequest, *apiresource.List[apiresource.LocationType]] {
	return &apiendpoint.APIEndpoint[*ListLocationTypesRequest, *apiresource.List[apiresource.LocationType]]{
		Title:             "List Location Types",
		Description:       "Returns a paginated list of all location types.",
		Method:            http.MethodGet,
		Route:             "/v1/operations/location-types",
		Request:           &ListLocationTypesRequest{},
		Response:          &apiresource.List[apiresource.LocationType]{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListLocationTypesRequest) (*apiresource.List[apiresource.LocationType], *apierror.APIError) {
			return svc.(LocationSvc).ListLocationTypes
		},
	}
}
