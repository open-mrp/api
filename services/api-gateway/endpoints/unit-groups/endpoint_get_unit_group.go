package unitgroupep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// GetUnitGroupRequest is a request to retrieve a unit group.
type GetUnitGroupRequest struct {
	// Unit group ID.
	UnitGroupID string `path:"id" validate:"required"`
}

type GetUnitGroupEndpoint struct{}

func (e *GetUnitGroupEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetUnitGroupRequest, *apiresource.UnitGroup] {
	return &apiendpoint.APIEndpoint[*GetUnitGroupRequest, *apiresource.UnitGroup]{
		Title:             "Get Unit Group",
		Description:       "Returns a unit group by ID.",
		Method:            http.MethodGet,
		Route:             "/v1/catalog/unit-groups/{id}",
		ContentType:       "application/json",
		Request:           &GetUnitGroupRequest{},
		Response:          &apiresource.UnitGroup{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetUnitGroupRequest) (*apiresource.UnitGroup, *apierror.APIError) {
			return svc.(UnitGroupSvc).GetUnitGroup
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeUnitGroup,
			Fields:     []string{"owner", "owner.account", "base_unit", "associated_units"},
		}),
	}
}
