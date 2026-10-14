package territoryep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
	"github.com/open-mrp/apikit/object"
)

// Request to delete a territory.
type DeleteTerritoryRequest struct {
	// ID of your account, which owns the territory.
	//
	// A path naming any other account returns a not-found error.
	AccountID string `path:"account_id" validate:"required"`
	// ID of the territory to delete.
	TerritoryID string `path:"id" validate:"required"`
}

// Deletes a territory.
//
// Sales orders that were already assigned a sales rep through this territory keep that rep; only later auto-assignment is affected. Deleting a territory that was already deleted returns an already-deleted error rather than a not-found error.
type DeleteTerritoryEndpoint struct{}

func (e *DeleteTerritoryEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteTerritoryRequest, *object.Deleted] {
	return (&apiendpoint.APIEndpoint[*DeleteTerritoryRequest, *object.Deleted]{
		Title:             "Delete Territory",
		Method:            http.MethodDelete,
		Route:             "/v1/sales/accounts/{account_id}/territories/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		AgentTool:         true,
		Preview:           true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainTerritories, Action: types.ActionDelete}}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteTerritoryRequest) (*object.Deleted, *apierror.APIError) {
			return apiresource.Deleted(svc.(TerritorySvc).DeleteTerritory, constants.ObjectTypeTerritory, "id")
		},
	})
}
