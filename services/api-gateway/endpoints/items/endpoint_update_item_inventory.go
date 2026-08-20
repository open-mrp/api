package itemep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apirequest "github.com/augno/api/services/api-gateway/pkg/request"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// Request to adjust or reconcile inventory for an item.
type UpdateItemInventoryRequest struct {
	// Item ID.
	ItemID string `path:"id" validate:"required"`
	// The quantity to apply, interpreted according to `operation`.
	//
	// With `adjust` it is added to the current quantity and may be negative; with `reconcile` the current quantity is set to exactly this value. It is recorded in the unit you send it in, and the current quantity a `reconcile` measures against is read in that same unit, so a reconcile to the figure already reported moves no stock.
	Quantity apirequest.QuantityInput `json:"quantity" validate:"required"`
	// How `quantity` is applied.
	//
	// - `adjust`: adds `quantity` to the current quantity.
	// - `reconcile`: sets the current quantity to exactly `quantity`.
	Operation field.Optional[constants.InventoryUpdateOperation] `json:"operation,omitzero"`
	// ID of the customer account that owns the resulting inventory.
	//
	// Use this for stock you hold but do not own, such as customer-supplied material. It only affects quantity being added: your account stays the holder, the customer becomes the owner, and the current quantity a `reconcile` measures against is still your account's. Requires edit access to that customer.
	CustomerID field.Optional[string] `json:"customer_id,omitzero" validate:"omitempty"`
	// ID of the location to record the inventory change against.
	//
	// Must be a location in your account.
	LocationID field.Optional[string] `json:"location_id,omitzero" validate:"omitempty"`
	// Lot number to record the inventory change against.
	//
	// The lot is created for the item if it does not already exist.
	LotNumber field.Optional[string] `json:"lot_number,omitzero" validate:"omitempty,max=255"`
}

var sampleUpdateItemInventoryRequest = &UpdateItemInventoryRequest{
	Quantity: apirequest.QuantityInput{
		Value:  "10.5",
		UnitID: apiresource.SampleUnitID,
	},
	Operation:  field.Some(constants.InventoryUpdateOperationAdjust),
	CustomerID: field.Some(apiresource.SampleCustomerID),
	LocationID: field.Some(apiresource.SampleLocationID),
}

func (*UpdateItemInventoryRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateItemInventoryRequest)
}

// Adjusts or reconciles the quantity of an item you hold.
//
// With `operation` set to `adjust` (the behavior when it is omitted), `quantity` is added to the current quantity; with `reconcile`, the current quantity is set to exactly `quantity`. Either way it is the resulting difference that gets written, so a difference of zero moves no stock.
//
// The figure a `reconcile` measures against is what is on hand net of demand nothing has covered — the same figure `available_to_promise` is derived from, not the raw on-hand total. Reconciling to the quantity already reported therefore writes nothing.
//
// Stock that arrives is allocated against unfilled demand for the item, so an adjustment can settle a shortfall instead of raising the quantity free to promise. That allocation happens just after the request rather than inside it, because it walks every open issue for the item. The change is recorded in the item's inventory audit trail as a user correction attributed to the caller.
type UpdateItemInventoryEndpoint struct{}

func (e *UpdateItemInventoryEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateItemInventoryRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*UpdateItemInventoryRequest, *apiresource.EmptyResource]{
		Title:               "Update Item Inventory",
		Method:              http.MethodPatch,
		Route:               "/v1/catalog/items/{id}/inventory",
		ContentType:         "application/json",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		Preview:             true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainItems, Action: types.ActionUpdate}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateItemInventoryRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(ItemSvc).UpdateItemInventory
		},
	})
}
