package unitgroupep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// GetUnitGroupUnitRequest is the request to retrieve a single unit group unit.
type GetUnitGroupUnitRequest struct {
	// The ID of the unit group.
	UnitGroupID string `path:"unitGroupId" validate:"required"`
	// The ID of the unit group unit to retrieve.
	UnitGroupUnitID string `path:"id" validate:"required"`
}

type GetUnitGroupUnitEndpoint struct{}

func (e *GetUnitGroupUnitEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetUnitGroupUnitRequest, *apiresource.UnitGroupUnit] {
	return &apiendpoint.APIEndpoint[*GetUnitGroupUnitRequest, *apiresource.UnitGroupUnit]{
		Title:             "Get Unit Group Unit",
		Description:       "Returns a single associated unit within a unit group by its ID.",
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
