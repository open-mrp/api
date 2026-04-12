package purchaseorderep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// UpdatePurchaseOrderRequest is the request to update a purchase order.
type UpdatePurchaseOrderRequest struct {
	// The ID of the purchase order to update.
	PurchaseOrderID string `path:"id" validate:"required"`
	// A note for the order.
	Note *string `json:"note,omitempty" nullable:"false"`
	// The purchase order number.
	Number *string `json:"number,omitempty" nullable:"false" validate:"omitempty,max=255"`
	// The priority code.
	PriorityCode *string `json:"priority_code,omitempty" nullable:"false" validate:"omitempty,max=255"`
	// The billing address ID.
	BillingAddressID *string `json:"billing_address_id,omitempty" nullable:"true" validate:"omitempty,max=191"`
	// The shipping address ID.
	ShippingAddressID *string `json:"shipping_address_id,omitempty" nullable:"true" validate:"omitempty,max=191"`
	// The promised/scheduled delivery date.
	PromisedAt *string `json:"promised_at,omitempty" nullable:"false"`
	// The account user IDs for email contacts (replaces existing).
	ContactAccountUserIDs []string `json:"contact_account_user_ids,omitempty"`
}

var sampleUpdatePONote = "Updated delivery notes"
var sampleUpdatePONumber = apiresource.SamplePurchaseOrderNumber
var sampleUpdatePOPriorityCode = string(constants.PriorityCodeNormal)
var sampleUpdatePOPromisedAt = "2026-05-15T00:00:00Z"
var sampleUpdatePurchaseOrderRequest = &UpdatePurchaseOrderRequest{
	Note:         &sampleUpdatePONote,
	Number:       &sampleUpdatePONumber,
	PriorityCode: &sampleUpdatePOPriorityCode,
	PromisedAt:   &sampleUpdatePOPromisedAt,
}

func (*UpdatePurchaseOrderRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdatePurchaseOrderRequest)
}

type UpdatePurchaseOrderEndpoint struct{}

func (e *UpdatePurchaseOrderEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdatePurchaseOrderRequest, *apiresource.PurchaseOrderDetail] {
	return &apiendpoint.APIEndpoint[*UpdatePurchaseOrderRequest, *apiresource.PurchaseOrderDetail]{
		Title:             "Update Purchase Order",
		Description:       "Partially updates a purchase order.",
		Method:            http.MethodPatch,
		ContentType:       "application/json",
		Route:             "/v1/operations/purchase-orders/{id}",
		Request:           &UpdatePurchaseOrderRequest{},
		Response:          &apiresource.PurchaseOrderDetail{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdatePurchaseOrderRequest) (*apiresource.PurchaseOrderDetail, *apierror.APIError) {
			return svc.(PurchaseOrderSvc).UpdatePurchaseOrder
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypePurchaseOrder,
			Fields:     []string{"supplier", "bill_to_address", "ship_to_address", "carrier", "service_level", "payment_term", "shipping_term", "receiving_order", "lines", "contacts"},
		}),
	}
}
