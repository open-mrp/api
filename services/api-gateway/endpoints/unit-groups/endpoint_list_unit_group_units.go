package unitgroupep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// ListUnitGroupUnitsRequest is the request to list units within a unit group.
type ListUnitGroupUnitsRequest struct {
	// The ID of the unit group.
	UnitGroupID string `path:"unitGroupId" validate:"required"`
}

type ListUnitGroupUnitsEndpoint struct{}

func (e *ListUnitGroupUnitsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListUnitGroupUnitsRequest, *apiresource.List[apiresource.UnitGroupUnit]] {
	return &apiendpoint.APIEndpoint[*ListUnitGroupUnitsRequest, *apiresource.List[apiresource.UnitGroupUnit]]{
		Title:             "List Unit Group Units",
		Description:       "Returns the list of associated units within the specified unit group.",
		Method:            http.MethodGet,
		Route:             "/v1/catalog/unit-groups/{unitGroupId}/units",
		Request:           &ListUnitGroupUnitsRequest{},
		Response:          &apiresource.List[apiresource.UnitGroupUnit]{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListUnitGroupUnitsRequest) (*apiresource.List[apiresource.UnitGroupUnit], *apierror.APIError) {
			return svc.(UnitGroupSvc).ListUnitGroupUnits
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeUnitGroupUnit,
			Fields:     []string{"unit"},
		}),
	}
}
