package inventoryep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list inventories.
type ListInventoriesRequest struct {
	apiresource.PaginationRequest
}

type ListInventoriesEndpoint struct{}

func (e *ListInventoriesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListInventoriesRequest, *apiresource.ListInventoriesResponse] {
	return &apiendpoint.APIEndpoint[*ListInventoriesRequest, *apiresource.ListInventoriesResponse]{
		Title:             "List Inventories",
		Description:       "Returns a paginated list of items with on-hand inventory quantities for the account.",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/inventories",
		Request:           &ListInventoriesRequest{},
		Response:          &apiresource.ListInventoriesResponse{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListInventoriesRequest) (*apiresource.ListInventoriesResponse, *apierror.APIError) {
			return svc.(InventorySvc).ListInventories
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeInventoryItem,
			Fields:     []string{"quantity.unit"},
		}),
	}
}
