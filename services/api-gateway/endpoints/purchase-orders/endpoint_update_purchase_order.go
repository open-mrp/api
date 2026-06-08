package purchaseorderep

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

// Request to update a purchase order.
type UpdatePurchaseOrderRequest struct {
	// Purchase order ID.
	PurchaseOrderID string `path:"id" validate:"required"`
	// Order note.
	Note field.Optional[string] `json:"note,omitzero"`
	// Purchase order number.
	Number field.Optional[string] `json:"number,omitzero" validate:"omitempty,max=255"`
	// Priority code.
	PriorityCode field.Optional[string] `json:"priority_code,omitzero" validate:"omitempty,max=255"`
	// Billing address ID.
	BillingAddressID field.Optional[string] `json:"billing_address_id,omitzero" validate:"omitempty"`
	// Shipping address ID.
	ShippingAddressID field.Optional[string] `json:"shipping_address_id,omitzero" validate:"omitempty"`
	// Promised delivery date.
	PromisedAt field.Optional[string] `json:"promised_at,omitzero"`
	// Account user IDs for email contacts. Replaces existing contacts.
	ContactAccountUserIDs field.Optional[[]string] `json:"contact_account_user_ids,omitzero"`
}

var sampleUpdatePONote = "Updated delivery notes"
var sampleUpdatePONumber = apiresource.SamplePurchaseOrderNumber
var sampleUpdatePOPriorityCode = string(constants.PriorityCodeNormal)
var sampleUpdatePOPromisedAt = "2026-05-15T00:00:00Z"
var sampleUpdatePurchaseOrderRequest = &UpdatePurchaseOrderRequest{
	Note:         field.Some(sampleUpdatePONote),
	Number:       field.Some(sampleUpdatePONumber),
	PriorityCode: field.Some(sampleUpdatePOPriorityCode),
	PromisedAt:   field.Some(sampleUpdatePOPromisedAt),
}

func (*UpdatePurchaseOrderRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdatePurchaseOrderRequest)
}

// Partially updates a purchase order.
type UpdatePurchaseOrderEndpoint struct{}

func (e *UpdatePurchaseOrderEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdatePurchaseOrderRequest, *apiresource.PurchaseOrder] {
	return (&apiendpoint.APIEndpoint[*UpdatePurchaseOrderRequest, *apiresource.PurchaseOrder]{
		Title:             "Update Purchase Order",
		Method:            http.MethodPatch,
		ContentType:       "application/json",
		Route:             "/v1/operations/purchase-orders/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypePurchaseOrder,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdatePurchaseOrderRequest) (*apiresource.PurchaseOrder, *apierror.APIError) {
			return svc.(PurchaseOrderSvc).UpdatePurchaseOrder
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypePurchaseOrder,
			Fields:     []string{"supplier", "bill_to_address", "ship_to_address", "freight", "payment_term", "shipping_term", "receiving_order", "lines", "contacts"},
		}),
	})
}
