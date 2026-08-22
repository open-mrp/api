package territoryep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to delete a territory.
type DeleteTerritoryRequest struct {
	// ID of your account, which owns the territory.
	AccountID string `path:"account_id" validate:"required"`
	// ID of the territory to delete.
	TerritoryID string `path:"id" validate:"required"`
}

// Deletes a territory.
//
// Sales orders that were already assigned a sales rep through this territory keep that rep; only later auto-assignment is affected. Deleting a territory that was already deleted returns an already-deleted error rather than a not-found error.
type DeleteTerritoryEndpoint struct{}

func (e *DeleteTerritoryEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteTerritoryRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*DeleteTerritoryRequest, *apiresource.EmptyResource]{
		Title:               "Delete Territory",
		Method:              http.MethodDelete,
		Route:               "/v1/sales/accounts/{account_id}/territories/{id}",
		ContentType:         "application/json",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		Preview:             true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainTerritories, Action: types.ActionDelete}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteTerritoryRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(TerritorySvc).DeleteTerritory
		},
	})
}
