package inventoryep

import (
	"context"
	"net/http"
	"time"

	"github.com/open-mrp/api/services/auth-service/pkg/types"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to list inventories.
type ListInventoriesRequest struct {
	apiresource.PaginationRequest
	// Reports each item's stock as it stood at this instant instead of now.
	//
	// The figure is then the last inventory level logged for the item at or before `as_of`: its physical stock — on hand less what was short against open demand — when the movement that wrote it happened. An item with nothing logged by then reports zero.
	AsOf *time.Time `query:"as_of"`
}

// Returns a paginated list of items with inventory quantities for the account.
//
// Items are listed whether or not they have ever held stock; an item with no recorded inventory reports a zero quantity. Items backed by a non-sale product — the service, shipping, tax, credit, and return products that carry charges on orders — are left out. The `q` search term matches on item SKU and description.
//
// Without `as_of`, each quantity is the item's current on-hand stock. With it, each is the item's last logged level at or before that instant, which is what an inventory valuation at a past date reads. Either way the figure is in the base unit of the item's category.
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
			Fields:     []string{"product_line"},
		}),
	})
}
