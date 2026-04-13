package territoryep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to retrieve a territory.
type GetTerritoryRequest struct {
	// Account ID.
	AccountID string `path:"account_id" validate:"required"`
	// Territory ID.
	TerritoryID string `path:"id" validate:"required"`
}

type GetTerritoryEndpoint struct{}

func (e *GetTerritoryEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetTerritoryRequest, *apiresource.Territory] {
	return &apiendpoint.APIEndpoint[*GetTerritoryRequest, *apiresource.Territory]{
		Title:             "Get Territory",
		Description:       "Returns a territory by ID.",
		Method:            http.MethodGet,
		Route:             "/v1/sales/accounts/{account_id}/territories/{id}",
		ContentType:       "application/json",
		Request:           &GetTerritoryRequest{},
		Response:          &apiresource.Territory{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetTerritoryRequest) (*apiresource.Territory, *apierror.APIError) {
			return svc.(TerritorySvc).GetTerritory
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeTerritory,
			Fields:     []string{"sales_rep", "product_line"},
		}),
	}
}
