package unitgroupep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// DeleteUnitGroupRequest is a request to delete a unit group.
type DeleteUnitGroupRequest struct {
	// Unit group ID.
	UnitGroupID string `path:"id" validate:"required"`
}

// Deletes a unit group and all associated unit conversions. System unit groups cannot be deleted.
type DeleteUnitGroupEndpoint struct{}

func (e *DeleteUnitGroupEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteUnitGroupRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*DeleteUnitGroupRequest, *apiresource.EmptyResource]{
		Title:             "Delete Unit Group",
		Method:            http.MethodDelete,
		Route:             "/v1/catalog/unit-groups/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteUnitGroupRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(UnitGroupSvc).DeleteUnitGroup
		},
	})
}
