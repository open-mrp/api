package unitep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to delete a unit.
type DeleteUnitRequest struct {
	// Unit ID.
	UnitID string `path:"id" validate:"required"`
}

type DeleteUnitEndpoint struct{}

func (e *DeleteUnitEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteUnitRequest, *apiresource.EmptyResource] {
	return &apiendpoint.APIEndpoint[*DeleteUnitRequest, *apiresource.EmptyResource]{
		Title:             "Delete Unit",
		Description:       "Deletes an account-owned unit. Associated unit group memberships are also removed, and system units cannot be deleted.",
		Method:            http.MethodDelete,
		Route:             "/v1/catalog/units/{id}",
		ContentType:       "application/json",
		Request:           &DeleteUnitRequest{},
		Response:          &apiresource.EmptyResource{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteUnitRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(UnitSvc).DeleteUnit
		},
	}
}
