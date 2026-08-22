package edidclocationep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to list DC locations.
type ListDCLocationsRequest struct {
	apiresource.PaginationRequest
}

// Returns a paginated list of DC locations for the target account.
//
// Locations are ordered by creation time, newest first. The `q` search term matches the location text and the name of the customer the location belongs to.
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
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainEdiRuns, Action: types.ActionRead},
		},
		ObjectType: constants.ObjectTypeDCLocation,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListDCLocationsRequest) (*apiresource.List[apiresource.DCLocation], *apierror.APIError) {
			return svc.(EDIDCLocationSvc).ListDCLocations
		},
	})
}
