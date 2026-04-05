package unitgroupep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// DeleteUnitGroupUnitRequest is the request to delete an associated unit from a unit group.
type DeleteUnitGroupUnitRequest struct {
	// The ID of the unit group.
	UnitGroupID string `path:"unitGroupId" validate:"required"`
	// The ID of the associated unit to delete.
	AssociatedUnitID string `path:"id" validate:"required"`
}

type DeleteUnitGroupUnitEndpoint struct{}

func (e *DeleteUnitGroupUnitEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteUnitGroupUnitRequest, *apiresource.EmptyResource] {
	return &apiendpoint.APIEndpoint[*DeleteUnitGroupUnitRequest, *apiresource.EmptyResource]{
		Title:             "Delete Unit Group Associated Unit",
		Description:       "Deletes an associated unit from a unit group.",
		Method:            http.MethodDelete,
		Route:             "/v1/catalog/unit-groups/{unitGroupId}/units/{id}",
		ContentType:       "application/json",
		Request:           &DeleteUnitGroupUnitRequest{},
		Response:          &apiresource.EmptyResource{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteUnitGroupUnitRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(UnitGroupSvc).DeleteUnitGroupUnit
		},
	}
}
