package unitgroupep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// DeleteUnitGroupUnitRequest is a request to delete an associated unit from a unit group.
type DeleteUnitGroupUnitRequest struct {
	// Unit group ID.
	UnitGroupID string `path:"unit_group_id" validate:"required"`
	// Unit group unit ID.
	AssociatedUnitID string `path:"id" validate:"required"`
}

// Deletes an associated unit from a unit group.
type DeleteUnitGroupUnitEndpoint struct{}

func (e *DeleteUnitGroupUnitEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteUnitGroupUnitRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*DeleteUnitGroupUnitRequest, *apiresource.EmptyResource]{
		Title:             "Delete Unit Group Associated Unit",
		Method:            http.MethodDelete,
		Route:             "/v1/catalog/unit-groups/{unit_group_id}/units/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteUnitGroupUnitRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(UnitGroupSvc).DeleteUnitGroupUnit
		},
	})
}
