package territoryep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list territories.
type ListTerritoriesRequest struct {
	// ID of your account, which owns the territories.
	AccountID string `path:"account_id" validate:"required"`
	apiresource.PaginationRequest
}

// Returns a paginated list of territories in your account, most recently created first.
//
// The `q` search term matches the state, the sales rep's name or email address, and the product line name.
type ListTerritoriesEndpoint struct{}

func (e *ListTerritoriesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListTerritoriesRequest, *apiresource.List[apiresource.Territory]] {
	return (&apiendpoint.APIEndpoint[*ListTerritoriesRequest, *apiresource.List[apiresource.Territory]]{
		Title:               "List Territories",
		Method:              http.MethodGet,
		ContentType:         "application/json",
		Route:               "/v1/sales/accounts/{account_id}/territories",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		Preview:             true,
		ObjectType:          constants.ObjectTypeTerritory,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainTerritories, Action: types.ActionRead}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListTerritoriesRequest) (*apiresource.List[apiresource.Territory], *apierror.APIError) {
			return svc.(TerritorySvc).ListTerritories
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeTerritory,
			Fields:     []string{"sales_rep", "sales_rep.user", "product_line"},
		}),
	})
}
