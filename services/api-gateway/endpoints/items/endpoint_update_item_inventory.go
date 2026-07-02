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

// UpdateItemInventoryRequest is the request to adjust or reconcile inventory for an item.
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
	// When provided, added inventory is recorded as owned by this customer account instead of your own; requires edit access to that customer.
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
// With `operation` set to `adjust` (the default), `quantity_change` is added to the current on-hand quantity; with `reconcile`, the on-hand quantity is set to exactly `quantity_change`. The change is recorded in the item's inventory audit trail as a user correction.
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
