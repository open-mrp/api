package salesorderep

import (
	"context"
	"net/http"
	"time"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// Request to update a sales order.
type UpdateSalesOrderRequest struct {
	// Sales order ID.
	SalesOrderID string `path:"id" validate:"required"`
	// Customer's purchase order number.
	CustomerPurchaseOrderNumber field.Optional[string] `json:"customer_purchase_order_number,omitzero" validate:"omitempty,max=255"`
	// Order note.
	Note field.Optional[string] `json:"note,omitzero"`
	// Carrier ID.
	CarrierID field.Optional[string] `json:"carrier_id,omitzero" validate:"omitempty"`
	// Service level ID.
	ServiceLevelID field.Optional[string] `json:"service_level_id,omitzero" validate:"omitempty"`
	// Who is billed for freight (sender or third_party).
	CarrierBillingType field.Optional[constants.CarrierBillingType] `json:"carrier_billing_type,omitzero" validate:"omitempty"`
	// Carrier billing account number.
	CarrierBillingAccountNumber field.Optional[string] `json:"carrier_billing_account_number,omitzero" validate:"omitempty,max=255"`
	// Priority code.
	PriorityCode field.Optional[string] `json:"priority_code,omitzero" validate:"omitempty,max=255"`
	// Sales rep ID.
	SalesRepID field.Optional[string] `json:"sales_rep_id,omitzero" validate:"omitempty"`
	// Shipping term ID.
	ShippingTermID field.Optional[string] `json:"shipping_term_id,omitzero" validate:"omitempty"`
	// Payment term ID.
	PaymentTermID field.Optional[string] `json:"payment_term_id,omitzero" validate:"omitempty"`
	// Order discount ID.
	OrderDiscountID field.Optional[string] `json:"order_discount_id,omitzero" validate:"omitempty"`
	// Billing address ID. Re-points the order to an existing address. To change
	// an address's contents, use the update-address endpoint.
	BillingAddressID field.Optional[string] `json:"billing_address_id,omitzero" validate:"omitempty"`
	// Shipping address ID. Re-points the order to an existing address. To change
	// an address's contents, use the update-address endpoint.
	ShippingAddressID field.Optional[string] `json:"shipping_address_id,omitzero" validate:"omitempty"`
	// Order number.
	Number field.Optional[string] `json:"number,omitzero" validate:"omitempty,max=255"`
	// Acknowledgment status (not_sent, sent).
	AcknowledgmentStatus field.Optional[constants.AcknowledgmentStatus] `json:"acknowledgment_status,omitzero" validate:"omitempty"`
	// Promised delivery date.
	PromisedAt field.Optional[time.Time] `json:"promised_at,omitzero"`
	// Customer ID.
	CustomerID field.Optional[string] `json:"customer_id,omitzero" validate:"omitempty"`
	// When set, replaces acknowledgement email contacts on the order.
	// An empty list clears all contacts; omitted leaves existing contacts untouched.
	AcknowledgementEmailContacts field.Optional[[]SalesOrderEmailContactInput] `json:"acknowledgement_email_contacts,omitzero"`
	// When set, replaces invoice email contacts on the order.
	// An empty list clears all contacts; omitted leaves existing contacts untouched.
	InvoiceEmailContacts field.Optional[[]SalesOrderEmailContactInput] `json:"invoice_email_contacts,omitzero"`
}

var sampleUpdateSONote = "Updated shipping instructions"
var sampleUpdateSOCarrierID = apiresource.SampleCarrierID
var sampleUpdateSOPriorityCode = string(constants.PriorityCodeNormal)
var sampleUpdateSOShippingAddressID = apiresource.SampleAddressID
var sampleUpdateSalesOrderRequest = &UpdateSalesOrderRequest{
	Note:              field.Some(sampleUpdateSONote),
	CarrierID:         field.Some(sampleUpdateSOCarrierID),
	PriorityCode:      field.Some(sampleUpdateSOPriorityCode),
	ShippingAddressID: field.Some(sampleUpdateSOShippingAddressID),
}

func (*UpdateSalesOrderRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateSalesOrderRequest)
}

// Partially updates a sales order.
type UpdateSalesOrderEndpoint struct{}

func (e *UpdateSalesOrderEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateSalesOrderRequest, *apiresource.SalesOrder] {
	return (&apiendpoint.APIEndpoint[*UpdateSalesOrderRequest, *apiresource.SalesOrder]{
		Title:             "Update Sales Order",
		Method:            http.MethodPatch,
		ContentType:       "application/json",
		Route:             "/v1/sales/sales-orders/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeSalesOrder,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateSalesOrderRequest) (*apiresource.SalesOrder, *apierror.APIError) {
			return svc.(SalesOrderSvc).UpdateSalesOrder
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeSalesOrder,
			Fields:     []string{"customer", "sales_rep", "bill_to_address", "ship_to_address", "freight", "payment_term", "shipping_term", "order_discount", "totals", "related.pick", "related.production_run", "related.shipments", "lines", "lines.product", "lines.quantity_ordered", "lines.unit_price", "lines.unit_cost", "lines.totals"},
		}),
	})
}
