package itemep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to retrieve an item's inventory position.
type RetrieveItemInventoryRequest struct {
	// Item ID.
	ItemID string `path:"id" validate:"required"`
}

// Returns the stock position for an item: what is on hand, what is reserved against existing orders, what is free to promise, and what is short.
//
// Stock your account either owns or holds counts toward the on-hand figure, so customer-supplied material sitting in your facility is included. All four quantities are reported in the base unit of the item's category.
type RetrieveItemInventoryEndpoint struct{}

func (e *RetrieveItemInventoryEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveItemInventoryRequest, *apiresource.ItemInventory] {
	return (&apiendpoint.APIEndpoint[*RetrieveItemInventoryRequest, *apiresource.ItemInventory]{
		Title:               "Retrieve Item Inventory",
		Method:              http.MethodGet,
		ContentType:         "application/json",
		Route:               "/v1/catalog/items/{id}/inventory",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainItems, Action: types.ActionRead}},
		Preview:             true,
		ObjectType:          constants.ObjectTypeItemInventory,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveItemInventoryRequest) (*apiresource.ItemInventory, *apierror.APIError) {
			return svc.(ItemSvc).GetItemInventory
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeItemInventory,
			Fields:     []string{"on_hand", "reserved", "available_to_promise", "short"},
		}),
	})
}
