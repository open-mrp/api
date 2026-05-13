package unitgroupep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// ListUnitGroupUnitsRequest is a request to list units within a unit group.
type ListUnitGroupUnitsRequest struct {
	// Unit group ID.
	UnitGroupID string `path:"unit_group_id" validate:"required"`
}

type ListUnitGroupUnitsEndpoint struct{}

func (e *ListUnitGroupUnitsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListUnitGroupUnitsRequest, *apiresource.List[apiresource.UnitGroupUnit]] {
	return &apiendpoint.APIEndpoint[*ListUnitGroupUnitsRequest, *apiresource.List[apiresource.UnitGroupUnit]]{
		Title:             "List Unit Group Units",
		Description:       "Returns a list of associated units within a unit group.",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/catalog/unit-groups/{unit_group_id}/units",
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
