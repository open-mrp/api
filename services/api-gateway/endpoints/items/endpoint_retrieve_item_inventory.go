package itemep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// RetrieveItemInventoryRequest is the request to get an item's inventory.
type RetrieveItemInventoryRequest struct {
	// Item ID.
	ItemID string `path:"id" validate:"required"`
}

// Returns inventory quantities for an item, including on-hand, reserved, available-to-promise, and short amounts.
type RetrieveItemInventoryEndpoint struct{}

func (e *RetrieveItemInventoryEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveItemInventoryRequest, *apiresource.ItemInventory] {
	return (&apiendpoint.APIEndpoint[*RetrieveItemInventoryRequest, *apiresource.ItemInventory]{
		Title:             "Retrieve Item Inventory",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/catalog/items/{id}/inventory",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveItemInventoryRequest) (*apiresource.ItemInventory, *apierror.APIError) {
			return svc.(ItemSvc).GetItemInventory
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeItemInventory,
			Fields:     []string{"on_hand", "reserved", "available_to_promise", "short"},
		}),
	})
}
