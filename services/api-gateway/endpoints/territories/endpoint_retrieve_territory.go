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
type RetrieveTerritoryRequest struct {
	// Account ID.
	AccountID string `path:"account_id" validate:"required"`
	// Territory ID.
	TerritoryID string `path:"id" validate:"required"`
}

type RetrieveTerritoryEndpoint struct{}

func (e *RetrieveTerritoryEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveTerritoryRequest, *apiresource.Territory] {
	return &apiendpoint.APIEndpoint[*RetrieveTerritoryRequest, *apiresource.Territory]{
		Title:             "Retrieve Territory",
		Description:       "Returns a territory by ID.",
		Method:            http.MethodGet,
		Route:             "/v1/sales/accounts/{account_id}/territories/{id}",
		ContentType:       "application/json",
		Request:           &RetrieveTerritoryRequest{},
		Response:          &apiresource.Territory{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveTerritoryRequest) (*apiresource.Territory, *apierror.APIError) {
			return svc.(TerritorySvc).GetTerritory
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeTerritory,
			Fields:     []string{"sales_rep", "product_line"},
		}),
	}
}
