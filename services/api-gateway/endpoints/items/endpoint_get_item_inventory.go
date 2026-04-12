package itemep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// GetItemInventoryRequest is the request to retrieve inventory for an item.
type GetItemInventoryRequest struct {
	// The ID of the item to retrieve inventory for.
	ItemID string `path:"id" validate:"required"`
}

type GetItemInventoryEndpoint struct{}

func (e *GetItemInventoryEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetItemInventoryRequest, *apiresource.ItemInventory] {
	return &apiendpoint.APIEndpoint[*GetItemInventoryRequest, *apiresource.ItemInventory]{
		Title:             "Get Item Inventory",
		Description:       "Returns inventory quantities for an item, including on-hand, reserved, available-to-promise, and short amounts.",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/catalog/items/{id}/inventory",
		Request:           &GetItemInventoryRequest{},
		Response:          &apiresource.ItemInventory{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetItemInventoryRequest) (*apiresource.ItemInventory, *apierror.APIError) {
			return svc.(ItemSvc).GetItemInventory
		},
	}
}
