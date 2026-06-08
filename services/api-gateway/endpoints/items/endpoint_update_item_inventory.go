package itemep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// UpdateItemInventoryRequest is the request to adjust or reconcile inventory for an item.
type UpdateItemInventoryRequest struct {
	// Item ID.
	ItemID string `path:"id" validate:"required"`
	// Quantity change to apply.
	QuantityChange field.Optional[float64] `json:"quantity_change,omitzero"`
	// How quantity_change is applied: adjust adds to current inventory; reconcile sets inventory to the exact value.
	Operation field.Optional[constants.InventoryUpdateOperation] `json:"operation,omitzero"`
	// Customer ID.
	CustomerID field.Optional[string] `json:"customer_id,omitzero" validate:"omitempty"`
	// Location ID.
	LocationID field.Optional[string] `json:"location_id,omitzero" validate:"omitempty"`
	// Lot number.
	LotNumber field.Optional[string] `json:"lot_number,omitzero" validate:"omitempty,max=255"`
	// Unit ID for the quantity change.
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

// Adjusts or reconciles inventory for an item. When operation is reconcile, inventory is set to the exact value; when operation is adjust, the quantity change is added to the current inventory.
type UpdateItemInventoryEndpoint struct{}

func (e *UpdateItemInventoryEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateItemInventoryRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*UpdateItemInventoryRequest, *apiresource.EmptyResource]{
		Title:             "Update Item Inventory",
		Method:            http.MethodPatch,
		Route:             "/v1/catalog/items/{id}/inventory",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateItemInventoryRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(ItemSvc).UpdateItemInventory
		},
	})
}
