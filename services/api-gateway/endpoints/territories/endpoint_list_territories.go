package territoryep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// ListTerritoriesRequest is the request to list territories for an account.
type ListTerritoriesRequest struct {
	// The ID of the account to list territories for.
	AccountID string `path:"account_id" validate:"required"`
	apiresource.PaginationRequest
}

type ListTerritoriesEndpoint struct{}

func (e *ListTerritoriesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListTerritoriesRequest, *apiresource.List[apiresource.Territory]] {
	return &apiendpoint.APIEndpoint[*ListTerritoriesRequest, *apiresource.List[apiresource.Territory]]{
		Title:             "List Territories",
		Description:       "Returns a paginated list of territories for the specified account.",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/sales/accounts/{account_id}/territories",
		Request:           &ListTerritoriesRequest{},
		Response:          &apiresource.List[apiresource.Territory]{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListTerritoriesRequest) (*apiresource.List[apiresource.Territory], *apierror.APIError) {
			return svc.(TerritorySvc).ListTerritories
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeTerritory,
			Fields:     []string{"sales_rep", "product_line"},
		}),
	}
}
