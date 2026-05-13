package itemep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// UpdateItemInventoryRequest is the request to adjust or reconcile inventory for an item.
type UpdateItemInventoryRequest struct {
	// Item ID.
	ItemID string `path:"id" validate:"required"`
	// Quantity change to apply.
	QuantityChange *float64 `json:"quantity_change,omitempty" nullable:"false"`
	// How quantity_change is applied: adjust adds to current inventory; reconcile sets inventory to the exact value.
	Operation *constants.InventoryUpdateOperation `json:"operation,omitempty" nullable:"false"`
	// Customer ID.
	CustomerID *string `json:"customer_id,omitempty" nullable:"false" validate:"omitempty"`
	// Location ID.
	LocationID *string `json:"location_id,omitempty" nullable:"false" validate:"omitempty"`
	// Lot number.
	LotNumber *string `json:"lot_number,omitempty" nullable:"false" validate:"omitempty,max=255"`
	// Unit ID for the quantity change.
	UnitID *string `json:"unit_id,omitempty" nullable:"false" validate:"omitempty"`
}

var sampleUpdateItemInventoryRequest = &UpdateItemInventoryRequest{
	QuantityChange: func() *float64 { v := 10.5; return &v }(),
	Operation:      func() *constants.InventoryUpdateOperation { v := constants.InventoryUpdateOperationAdjust; return &v }(),
	CustomerID:     func() *string { s := apiresource.SampleCustomerID; return &s }(),
	LocationID:     func() *string { s := apiresource.SampleLocationID; return &s }(),
	UnitID:         func() *string { s := apiresource.SampleUnitID; return &s }(),
}

func (*UpdateItemInventoryRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateItemInventoryRequest)
}

type UpdateItemInventoryEndpoint struct{}

func (e *UpdateItemInventoryEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateItemInventoryRequest, *apiresource.EmptyResource] {
	return &apiendpoint.APIEndpoint[*UpdateItemInventoryRequest, *apiresource.EmptyResource]{
		Title:             "Update Item Inventory",
		Description:       "Adjusts or reconciles inventory for an item. When operation is reconcile, inventory is set to the exact value; when operation is adjust, the quantity change is added to the current inventory.",
		Method:            http.MethodPatch,
		Route:             "/v1/catalog/items/{id}/inventory",
		ContentType:       "application/json",
		Request:           &UpdateItemInventoryRequest{},
		Response:          &apiresource.EmptyResource{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateItemInventoryRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(ItemSvc).UpdateItemInventory
		},
	}
}
