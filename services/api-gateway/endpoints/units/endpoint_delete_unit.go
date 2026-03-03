package unitep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// DeleteUnitRequest is the request to delete a unit.
type DeleteUnitRequest struct {
	// The ID of the unit to delete.
	UnitID string `path:"id"`
}

const deleteUnitEndpointDescription string = `This endpoint deletes an account-owned unit.
Associated unit group memberships are also removed. System units cannot be deleted.`

type DeleteUnitEndpoint struct{}

func (e *DeleteUnitEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteUnitRequest, *apiresource.EmptyResource] {
	return &apiendpoint.APIEndpoint[*DeleteUnitRequest, *apiresource.EmptyResource]{
		Title:             "Delete Unit",
		Description:       deleteUnitEndpointDescription,
		Method:            http.MethodDelete,
		Route:             "/v1/core/units/{id}",
		ContentType:       "application/json",
		Request:           &DeleteUnitRequest{},
		Response:          &apiresource.EmptyResource{},
		SuccessStatusCode: http.StatusNoContent,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteUnitRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(UnitSvc).DeleteUnit
		},
		Extras: apiendpoint.APIEndpointExtras{
			AllowUnknownJSONFields: false,
		},
	}
}
