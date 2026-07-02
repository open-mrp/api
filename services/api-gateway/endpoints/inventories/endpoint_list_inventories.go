package inventoryep

import (
	"context"
	"net/http"

	"github.com/augno/api/services/auth-service/pkg/types"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list inventories.
type ListInventoriesRequest struct {
	apiresource.PaginationRequest
}

// Returns a paginated list of items with on-hand inventory quantities for the account.
//
// Every item in the account appears once; items with no recorded inventory report a zero quantity.
type ListInventoriesEndpoint struct{}

func (e *ListInventoriesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListInventoriesRequest, *apiresource.List[apiresource.InventoryItem]] {
	return (&apiendpoint.APIEndpoint[*ListInventoriesRequest, *apiresource.List[apiresource.InventoryItem]]{
		Title:             "List Inventories",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/inventories",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainItems, Action: types.ActionRead},
		},
		ObjectType: constants.ObjectTypeInventoryItem,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListInventoriesRequest) (*apiresource.List[apiresource.InventoryItem], *apierror.APIError) {
			return svc.(InventorySvc).ListInventories
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeInventoryItem,
			Fields:     []string{"quantity.unit"},
		}),
	})
}
