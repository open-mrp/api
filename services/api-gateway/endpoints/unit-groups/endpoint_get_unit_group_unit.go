package unitgroupep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// GetUnitGroupUnitRequest is a request to retrieve a unit group unit.
type GetUnitGroupUnitRequest struct {
	// Unit group ID.
	UnitGroupID string `path:"unitGroupId" validate:"required"`
	// Unit group unit ID.
	UnitGroupUnitID string `path:"id" validate:"required"`
}

type GetUnitGroupUnitEndpoint struct{}

func (e *GetUnitGroupUnitEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetUnitGroupUnitRequest, *apiresource.UnitGroupUnit] {
	return &apiendpoint.APIEndpoint[*GetUnitGroupUnitRequest, *apiresource.UnitGroupUnit]{
		Title:             "Get Unit Group Unit",
		Description:       "Returns an associated unit within a unit group by ID.",
		Method:            http.MethodGet,
		Route:             "/v1/catalog/unit-groups/{unitGroupId}/units/{id}",
		ContentType:       "application/json",
		Request:           &GetUnitGroupUnitRequest{},
		Response:          &apiresource.UnitGroupUnit{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetUnitGroupUnitRequest) (*apiresource.UnitGroupUnit, *apierror.APIError) {
			return svc.(UnitGroupSvc).GetUnitGroupUnit
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeUnitGroupUnit,
			Fields:     []string{"unit"},
		}),
	}
}
