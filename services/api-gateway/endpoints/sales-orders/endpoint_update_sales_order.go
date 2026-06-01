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
)

// Request to update a sales order.
type UpdateSalesOrderRequest struct {
	// Sales order ID.
	SalesOrderID string `path:"id" validate:"required"`
	// Customer purchase order number.
	CustomerPONumber *string `json:"customer_po_number,omitempty" validate:"omitempty,max=255"`
	// Order note.
	Note *string `json:"note,omitempty"`
	// Carrier ID.
	CarrierID *string `json:"carrier_id,omitempty" validate:"omitempty"`
	// Service level ID.
	ServiceLevelID *string `json:"service_level_id,omitempty" validate:"omitempty"`
	// Carrier billing type.
	CarrierBillingType *string `json:"carrier_billing_type,omitempty" validate:"omitempty,max=255"`
	// Carrier billing account number.
	CarrierBillingAccount *string `json:"carrier_billing_account,omitempty" validate:"omitempty,max=255"`
	// Priority code.
	PriorityCode *string `json:"priority_code,omitempty" validate:"omitempty,max=255"`
	// Sales rep ID.
	SalesRepID *string `json:"sales_rep_id,omitempty" validate:"omitempty"`
	// Shipping term ID.
	ShippingTermID *string `json:"shipping_term_id,omitempty" validate:"omitempty"`
	// Payment term ID.
	PaymentTermID *string `json:"payment_term_id,omitempty" validate:"omitempty"`
	// Order discount ID.
	OrderDiscountID *string `json:"order_discount_id,omitempty" validate:"omitempty"`
	// Bill-to address name.
	BillToName *string `json:"bill_to_name,omitempty" validate:"omitempty,max=255"`
	// Bill-to street line 1.
	BillToStreetLine1 *string `json:"bill_to_street_line_1,omitempty" validate:"omitempty,max=255"`
	// Bill-to street line 2.
	BillToStreetLine2 *string `json:"bill_to_street_line_2,omitempty" validate:"omitempty,max=255"`
	// Bill-to locality/city.
	BillToLocality *string `json:"bill_to_locality,omitempty" validate:"omitempty,max=255"`
	// Bill-to state/province.
	BillToState *string `json:"bill_to_state,omitempty" validate:"omitempty,max=255"`
	// Bill-to postal code.
	BillToPostalCode *string `json:"bill_to_postal_code,omitempty" validate:"omitempty,max=255"`
	// Bill-to country.
	BillToCountry *string `json:"bill_to_country,omitempty" validate:"omitempty,max=2"`
	// Ship-to address name.
	ShipToName *string `json:"ship_to_name,omitempty" validate:"omitempty,max=255"`
	// Ship-to street line 1.
	ShipToStreetLine1 *string `json:"ship_to_street_line_1,omitempty" validate:"omitempty,max=255"`
	// Ship-to street line 2.
	ShipToStreetLine2 *string `json:"ship_to_street_line_2,omitempty" validate:"omitempty,max=255"`
	// Ship-to locality/city.
	ShipToLocality *string `json:"ship_to_locality,omitempty" validate:"omitempty,max=255"`
	// Ship-to state/province.
	ShipToState *string `json:"ship_to_state,omitempty" validate:"omitempty,max=255"`
	// Ship-to postal code.
	ShipToPostalCode *string `json:"ship_to_postal_code,omitempty" validate:"omitempty,max=255"`
	// Ship-to country.
	ShipToCountry *string `json:"ship_to_country,omitempty" validate:"omitempty,max=2"`
	// Order number.
	Number *string `json:"number,omitempty" validate:"omitempty,max=255"`
	// Whether the acknowledgment has been sent.
	IsAcknowledgmentSent *bool `json:"is_acknowledgment_sent,omitempty"`
	// Promised delivery date.
	PromisedAt *time.Time `json:"promised_at,omitempty"`
	// Customer ID.
	CustomerID *string `json:"customer_id,omitempty" validate:"omitempty"`
	// When set, replaces acknowledgement email contacts on the order.
	// An empty list clears all contacts; omitted leaves existing contacts untouched.
	AcknowledgementEmailContacts *[]SalesOrderEmailContactInput `json:"acknowledgement_email_contacts,omitempty"`
	// When set, replaces invoice email contacts on the order.
	// An empty list clears all contacts; omitted leaves existing contacts untouched.
	InvoiceEmailContacts *[]SalesOrderEmailContactInput `json:"invoice_email_contacts,omitempty"`
}

var sampleUpdateSONote = "Updated shipping instructions"
var sampleUpdateSOCarrierID = apiresource.SampleCarrierID
var sampleUpdateSOPriorityCode = string(constants.PriorityCodeNormal)
var sampleUpdateSOShipToName = apiresource.SampleCustomerName
var sampleUpdateSOShipToStreetLine1 = apiresource.SampleAddressLine1
var sampleUpdateSalesOrderRequest = &UpdateSalesOrderRequest{
	Note:              &sampleUpdateSONote,
	CarrierID:         &sampleUpdateSOCarrierID,
	PriorityCode:      &sampleUpdateSOPriorityCode,
	ShipToName:        &sampleUpdateSOShipToName,
	ShipToStreetLine1: &sampleUpdateSOShipToStreetLine1,
}

func (*UpdateSalesOrderRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateSalesOrderRequest)
}

// Partially updates a sales order.
type UpdateSalesOrderEndpoint struct{}

func (e *UpdateSalesOrderEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateSalesOrderRequest, *apiresource.SalesOrderDetail] {
	return (&apiendpoint.APIEndpoint[*UpdateSalesOrderRequest, *apiresource.SalesOrderDetail]{
		Title:             "Update Sales Order",
		Method:            http.MethodPatch,
		ContentType:       "application/json",
		Route:             "/v1/sales/sales-orders/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeSalesOrder,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateSalesOrderRequest) (*apiresource.SalesOrderDetail, *apierror.APIError) {
			return svc.(SalesOrderSvc).UpdateSalesOrder
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeSalesOrder,
			Fields:     []string{"customer", "bill_to_address", "ship_to_address", "carrier", "service_level", "payment_term", "shipping_term", "order_discount", "lines", "lines.item"},
		}),
	})
}
