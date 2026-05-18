package unitgroupep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// RetrieveUnitGroupUnitRequest is a request to retrieve a unit group unit.
type RetrieveUnitGroupUnitRequest struct {
	// Unit group ID.
	UnitGroupID string `path:"unit_group_id" validate:"required"`
	// Unit group unit ID.
	UnitGroupUnitID string `path:"id" validate:"required"`
}

// Returns an associated unit within a unit group by ID.
type RetrieveUnitGroupUnitEndpoint struct{}

func (e *RetrieveUnitGroupUnitEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveUnitGroupUnitRequest, *apiresource.UnitGroupUnit] {
	return (&apiendpoint.APIEndpoint[*RetrieveUnitGroupUnitRequest, *apiresource.UnitGroupUnit]{
		Title:             "Retrieve Unit Group Unit",
		Method:            http.MethodGet,
		Route:             "/v1/catalog/unit-groups/{unit_group_id}/units/{id}",
		ContentType:       "application/json",
		Request:           &RetrieveUnitGroupUnitRequest{},
		Response:          &apiresource.UnitGroupUnit{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveUnitGroupUnitRequest) (*apiresource.UnitGroupUnit, *apierror.APIError) {
			return svc.(UnitGroupSvc).GetUnitGroupUnit
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeUnitGroupUnit,
			Fields:     []string{"unit"},
		}),
	}).WithDocSource(e)
}
