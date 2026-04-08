package itemep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// UpdateItemInventoryRequest is the request to adjust or reconcile inventory for an item.
type UpdateItemInventoryRequest struct {
	// The ID of the item to update inventory for.
	ItemID string `path:"id" validate:"required"`
	// The quantity change to apply.
	QuantityChange *float64 `json:"quantity_change,omitempty" nullable:"false"`
	// Whether to reconcile (force to exact value) or adjust (add delta).
	Reconcile *bool `json:"reconcile,omitempty"`
	// Optional customer to update inventory for.
	CustomerID *string `json:"customer_id,omitempty" validate:"omitempty,max=191"`
	// Optional location.
	LocationID *string `json:"location_id,omitempty" validate:"omitempty,max=191"`
	// Optional lot number.
	LotNumber *string `json:"lot_number,omitempty" validate:"omitempty,max=255"`
	// The unit ID for the quantity change.
	UnitID *string `json:"unit_id,omitempty" nullable:"false" validate:"omitempty,max=191"`
}

var sampleUpdateItemInventoryRequest = &UpdateItemInventoryRequest{
	QuantityChange: func() *float64 { v := 10.5; return &v }(),
	Reconcile:      func() *bool { v := false; return &v }(),
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
		Description:       "Adjusts or reconciles inventory for an item. In reconcile mode, inventory is set to the exact value; in adjust mode, the quantity change is added to the current inventory.",
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
