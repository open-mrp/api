package territoryep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// DeleteTerritoryRequest is the request to delete a territory.
type DeleteTerritoryRequest struct {
	// The ID of the account that owns the territory.
	AccountID string `path:"account_id" validate:"required"`
	// The ID of the territory to delete.
	TerritoryID string `path:"id" validate:"required"`
}

type DeleteTerritoryEndpoint struct{}

func (e *DeleteTerritoryEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteTerritoryRequest, *apiresource.EmptyResource] {
	return &apiendpoint.APIEndpoint[*DeleteTerritoryRequest, *apiresource.EmptyResource]{
		Title:             "Delete Territory",
		Description:       "Deletes a territory from the specified account.",
		Method:            http.MethodDelete,
		Route:             "/v1/sales/accounts/{account_id}/territories/{id}",
		ContentType:       "application/json",
		Request:           &DeleteTerritoryRequest{},
		Response:          &apiresource.EmptyResource{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteTerritoryRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(TerritorySvc).DeleteTerritory
		},
	}
}
