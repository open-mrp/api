package purchaseorderep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/field"
)

// Request to update a purchase order.
type UpdatePurchaseOrderRequest struct {
	// Purchase order ID.
	PurchaseOrderID string `path:"id" validate:"required"`
	// Free-form note to record on the order.
	Note field.Optional[string] `json:"note,omitzero"`
	// New purchase order number, replacing the one assigned at creation.
	//
	// Must be unique within the account; a number already used by another order is rejected.
	Number field.Optional[string] `json:"number,omitzero" validate:"omitempty,max=255"`
	// Priority level for fulfilling the order (`low`, `normal`, or `high`).
	PriorityCode field.Optional[constants.PriorityCode] `json:"priority_code,omitzero" validate:"omitempty"`
	// ID of an existing address to use as the bill-to address.
	BillingAddressID field.Optional[string] `json:"billing_address_id,omitzero" validate:"omitempty"`
	// ID of an existing address to use as the ship-to address.
	ShippingAddressID field.Optional[string] `json:"shipping_address_id,omitzero" validate:"omitempty"`
	// Promised delivery date in `YYYY-MM-DD` format.
	//
	// Returned as `scheduled_at` on the purchase order resource.
	PromisedAt field.Optional[string] `json:"promised_at,omitzero"`
	// IDs of account users to set as the order's email contacts.
	//
	// Replaces the full set of existing contacts; omit the field to leave contacts unchanged.
	ContactAccountUserIDs field.Optional[[]string] `json:"contact_account_user_ids,omitzero"`
}

var sampleUpdatePONote = "Updated delivery notes"
var sampleUpdatePONumber = apiresource.SamplePurchaseOrderNumber
var sampleUpdatePOPriorityCode = constants.PriorityCodeNormal
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
//
// Only the fields sent are changed. Addresses are repointed at existing address records here, unlike create, which builds new addresses from inline fields; the order's lifecycle status is changed through the change-status endpoint instead.
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
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainPurchaseOrders, Action: types.ActionUpdate},
		},
		ObjectType: constants.ObjectTypePurchaseOrder,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdatePurchaseOrderRequest) (*apiresource.PurchaseOrder, *apierror.APIError) {
			return svc.(PurchaseOrderSvc).UpdatePurchaseOrder
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypePurchaseOrder,
			Fields:     []string{"supplier", "bill_to_address", "ship_to_address", "freight", "payment_term", "shipping_term", "receiving_order", "lines", "contacts"},
		}),
	})
}
