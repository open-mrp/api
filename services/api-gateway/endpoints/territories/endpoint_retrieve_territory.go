package territoryep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to retrieve a territory.
type RetrieveTerritoryRequest struct {
	// ID of your account, which owns the territory.
	AccountID string `path:"account_id" validate:"required"`
	// ID of the territory to retrieve.
	TerritoryID string `path:"id" validate:"required"`
}

// Returns a territory by ID.
type RetrieveTerritoryEndpoint struct{}

func (e *RetrieveTerritoryEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveTerritoryRequest, *apiresource.Territory] {
	return (&apiendpoint.APIEndpoint[*RetrieveTerritoryRequest, *apiresource.Territory]{
		Title:               "Retrieve Territory",
		Method:              http.MethodGet,
		Route:               "/v1/sales/accounts/{account_id}/territories/{id}",
		ContentType:         "application/json",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		Preview:             true,
		ObjectType:          constants.ObjectTypeTerritory,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainTerritories, Action: types.ActionRead}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveTerritoryRequest) (*apiresource.Territory, *apierror.APIError) {
			return svc.(TerritorySvc).GetTerritory
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeTerritory,
			Fields:     []string{"sales_rep", "sales_rep.user", "product_line"},
		}),
	})
}
