package unitep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// ListUnitsRequest is the request to list units with optional filters.
type ListUnitsRequest struct {
	apiresource.PaginationRequest
	// Filter by unit dimension code (e.g. "mass", "quantity").
	Type *string `query:"type"`
	// Filter by unit group membership.
	UnitGroupIDs []string `query:"unit_group_ids"`
}

const listUnitsEndpointDescription string = `This endpoint returns a paginated list of units for the target account, including both account-specific and global system units.
Supports cursor-based pagination, filtering by dimension type and unit group membership, and search by name or abbreviation.`

type ListUnitsEndpoint struct{}

func (e *ListUnitsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListUnitsRequest, *apiresource.List[apiresource.Unit]] {
	return &apiendpoint.APIEndpoint[*ListUnitsRequest, *apiresource.List[apiresource.Unit]]{
		Title:             "List Units",
		Description:       listUnitsEndpointDescription,
		Method:            http.MethodGet,
		Route:             "/v1/core/units",
		Request:           &ListUnitsRequest{},
		Response:          &apiresource.List[apiresource.Unit]{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListUnitsRequest) (*apiresource.List[apiresource.Unit], *apierror.APIError) {
			return svc.(UnitSvc).ListUnits
		},
		Extras: apiendpoint.APIEndpointExtras{
			AllowUnknownJSONFields: false,
		},
	}
}
