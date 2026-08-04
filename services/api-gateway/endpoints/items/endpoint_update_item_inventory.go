package itemep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
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
	// With `adjust`, it is added to the current on-hand quantity and may be negative; with `reconcile`, the on-hand quantity is set to exactly this value.
	QuantityChange field.Optional[float64] `json:"quantity_change,omitzero"`
	// How `quantity_change` is applied.
	//
	// - `adjust`: adds `quantity_change` to the current on-hand quantity.
	// - `reconcile`: sets the on-hand quantity to exactly `quantity_change`.
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
	// ID of the unit `quantity_change` is expressed in.
	//
	// The figure is recorded exactly as sent, with no conversion, so send it in the base unit of the item's category to keep it comparable with the quantities the inventory endpoints report.
	UnitID field.Optional[string] `json:"unit_id,omitzero" validate:"omitempty"`
}

var sampleUpdateItemInventoryRequest = &UpdateItemInventoryRequest{
	QuantityChange: field.Some(10.5),
	Operation:      field.Some(constants.InventoryUpdateOperationAdjust),
	CustomerID:     field.Some(apiresource.SampleCustomerID),
	LocationID:     field.Some(apiresource.SampleLocationID),
	UnitID:         field.Some(apiresource.SampleUnitID),
}

func (*UpdateItemInventoryRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateItemInventoryRequest)
}

// Adjusts or reconciles on-hand inventory for an item.
//
// With `operation` set to `adjust` (the behavior when it is omitted), `quantity_change` is added to the current on-hand quantity; with `reconcile`, the on-hand quantity is set to exactly `quantity_change`. Either way it is the resulting difference that gets written, so a difference of zero moves no stock.
//
// Added stock is immediately allocated against any unfilled demand for the item, so an adjustment can settle a shortfall instead of raising the quantity free on hand. The change is recorded in the item's inventory audit trail as a user correction attributed to the caller.
type UpdateItemInventoryEndpoint struct{}

func (e *UpdateItemInventoryEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateItemInventoryRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*UpdateItemInventoryRequest, *apiresource.EmptyResource]{
		Title:               "Update Item Inventory",
		Method:              http.MethodPatch,
		Route:               "/v1/catalog/items/{id}/inventory",
		ContentType:         "application/json",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		Preview:             true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainItems, Action: types.ActionUpdate}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateItemInventoryRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(ItemSvc).UpdateItemInventory
		},
	})
}
