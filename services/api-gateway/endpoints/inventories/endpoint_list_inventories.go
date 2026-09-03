package inventoryep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/auth-service/pkg/types"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to list inventories.
type ListInventoriesRequest struct {
	apiresource.PaginationRequest
}

// Returns a paginated list of items with on-hand inventory quantities for the account.
//
// Items are listed whether or not they have ever held stock; an item with no recorded inventory reports a zero quantity. Items backed by a non-sale product — the service, shipping, tax, credit, and return products that carry charges on orders — are left out. The `q` search term matches on item SKU and description.
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
	})
}
