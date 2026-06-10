package salesorderep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apirequest "github.com/augno/api/services/api-gateway/pkg/request"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// Request to create a sales order.
type CreateSalesOrderRequest struct {
	// Buyer account ID.
	BuyerAccountID string `json:"buyer_account_id" validate:"required"`
	// Customer's purchase order number.
	CustomerPurchaseOrderNumber field.Optional[string] `json:"customer_purchase_order_number,omitzero" validate:"omitempty,max=255"`
	// Order note.
	Note field.Optional[string] `json:"note,omitzero"`
	// Carrier ID.
	CarrierID field.Optional[string] `json:"carrier_id,omitzero" validate:"omitempty"`
	// Service level ID.
	ServiceLevelID field.Optional[string] `json:"service_level_id,omitzero" validate:"omitempty"`
	// Who is billed for freight.
	CarrierBillingType field.Optional[constants.CarrierBillingType] `json:"carrier_billing_type,omitzero" validate:"omitempty"`
	// Carrier billing account number.
	CarrierBillingAccountNumber field.Optional[string] `json:"carrier_billing_account_number,omitzero" validate:"omitempty,max=255"`
	// Priority code.
	PriorityCode string `json:"priority_code" validate:"required,max=255"`
	// Sales rep ID.
	SalesRepID field.Optional[string] `json:"sales_rep_id,omitzero" validate:"omitempty"`
	// Shipping term ID.
	ShippingTermID field.Optional[string] `json:"shipping_term_id,omitzero" validate:"omitempty"`
	// Sales order type code.
	SalesOrderTypeCode string `json:"sales_order_type_code" validate:"required,max=255"`
	// Payment term ID.
	PaymentTermID field.Optional[string] `json:"payment_term_id,omitzero" validate:"omitempty"`
	// Order discount ID.
	OrderDiscountID field.Optional[string] `json:"order_discount_id,omitzero" validate:"omitempty"`
	// Bill-to address name.
	BillToName field.Optional[string] `json:"bill_to_name,omitzero" validate:"omitempty,max=255"`
	// Bill-to street line 1.
	BillToStreetLine1 field.Optional[string] `json:"bill_to_street_line_1,omitzero" validate:"omitempty,max=255"`
	// Bill-to street line 2.
	BillToStreetLine2 field.Optional[string] `json:"bill_to_street_line_2,omitzero" validate:"omitempty,max=255"`
	// Bill-to locality/city.
	BillToLocality field.Optional[string] `json:"bill_to_locality,omitzero" validate:"omitempty,max=255"`
	// Bill-to state/province.
	BillToState field.Optional[string] `json:"bill_to_state,omitzero" validate:"omitempty,max=255"`
	// Bill-to postal code.
	BillToPostalCode field.Optional[string] `json:"bill_to_postal_code,omitzero" validate:"omitempty,max=255"`
	// Bill-to country.
	BillToCountry field.Optional[string] `json:"bill_to_country,omitzero" validate:"omitempty,max=2"`
	// Ship-to address name.
	ShipToName field.Optional[string] `json:"ship_to_name,omitzero" validate:"omitempty,max=255"`
	// Ship-to street line 1.
	ShipToStreetLine1 field.Optional[string] `json:"ship_to_street_line_1,omitzero" validate:"omitempty,max=255"`
	// Ship-to street line 2.
	ShipToStreetLine2 field.Optional[string] `json:"ship_to_street_line_2,omitzero" validate:"omitempty,max=255"`
	// Ship-to locality/city.
	ShipToLocality field.Optional[string] `json:"ship_to_locality,omitzero" validate:"omitempty,max=255"`
	// Ship-to state/province.
	ShipToState field.Optional[string] `json:"ship_to_state,omitzero" validate:"omitempty,max=255"`
	// Ship-to postal code.
	ShipToPostalCode field.Optional[string] `json:"ship_to_postal_code,omitzero" validate:"omitempty,max=255"`
	// Ship-to country.
	ShipToCountry field.Optional[string] `json:"ship_to_country,omitzero" validate:"omitempty,max=2"`
	// Order lines to create.
	Lines []CreateSalesOrderLineInput `json:"lines"`
	// Account users who should receive order acknowledgement emails.
	AcknowledgementEmailContacts []SalesOrderEmailContactInput `json:"acknowledgement_email_contacts,omitzero"`
	// Account users who should receive invoice emails.
	InvoiceEmailContacts []SalesOrderEmailContactInput `json:"invoice_email_contacts,omitzero"`
}

// Line item input for a create sales order request.
type CreateSalesOrderLineInput struct {
	apirequest.OrderLineInput
}

// SalesOrderEmailContactInput represents an account user subscribed to a sales-order email notification type.
type SalesOrderEmailContactInput struct {
	// Account user ID to receive the notification.
	AccountUserID string `json:"account_user_id" validate:"required"`
}

var sampleCreateSONote = "Rush order for trade show"
var sampleCreateSOCarrierID = apiresource.SampleCarrierID
var sampleCreateSOServiceLevelID = apiresource.SampleServiceLevelID
var sampleCreateSOShipToName = apiresource.SampleCustomerName
var sampleCreateSOShipToStreetLine1 = apiresource.SampleAddressLine1
var sampleCreateSOShipToLocality = apiresource.SampleAddressCity
var sampleCreateSOShipToState = apiresource.SampleAddressState
var sampleCreateSOShipToPostalCode = apiresource.SampleAddressPostalCode
var sampleCreateSOShipToCountry = apiresource.SampleAddressCountry
var sampleCreateSalesOrderRequest = &CreateSalesOrderRequest{
	BuyerAccountID:     apiresource.SampleCustomerID,
	Note:               field.Some(sampleCreateSONote),
	CarrierID:          field.Some(sampleCreateSOCarrierID),
	ServiceLevelID:     field.Some(sampleCreateSOServiceLevelID),
	PriorityCode:       string(constants.PriorityCodeNormal),
	SalesOrderTypeCode: "sales_order",
	ShipToName:         field.Some(sampleCreateSOShipToName),
	ShipToStreetLine1:  field.Some(sampleCreateSOShipToStreetLine1),
	ShipToLocality:     field.Some(sampleCreateSOShipToLocality),
	ShipToState:        field.Some(sampleCreateSOShipToState),
	ShipToPostalCode:   field.Some(sampleCreateSOShipToPostalCode),
	ShipToCountry:      field.Some(sampleCreateSOShipToCountry),
	Lines: []CreateSalesOrderLineInput{
		{
			OrderLineInput: apirequest.OrderLineInput{
				ProductID:                  apiresource.SampleProductID,
				ProductSKU:                 "WIDGET-001",
				QuantityValue:              "10",
				QuantityUnitID:             apiresource.SampleUnitID,
				UnitPriceValue:             "25.00",
				UnitPriceNumeratorUnitID:   apiresource.SampleUnitID,
				UnitPriceDenominatorUnitID: apiresource.SampleUnitID,
			},
		},
	},
}

func (*CreateSalesOrderRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateSalesOrderRequest)
}

// Creates a sales order.
type CreateSalesOrderEndpoint struct{}

func (e *CreateSalesOrderEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateSalesOrderRequest, *apiresource.SalesOrder] {
	return (&apiendpoint.APIEndpoint[*CreateSalesOrderRequest, *apiresource.SalesOrder]{
		Title:             "Create Sales Order",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/sales/sales-orders",
		SuccessStatusCode: http.StatusCreated,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeSalesOrder,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateSalesOrderRequest) (*apiresource.SalesOrder, *apierror.APIError) {
			return svc.(SalesOrderSvc).CreateSalesOrder
		},
		LocationFunc: func(resp *apiresource.SalesOrder) string {
			return "/v1/sales/sales-orders/" + resp.ID
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeSalesOrder,
			Fields:     []string{"customer", "sales_rep", "bill_to_address", "ship_to_address", "freight", "payment_term", "shipping_term", "order_discount", "totals", "related.pick", "related.production_run", "related.shipments", "lines", "lines.product", "lines.quantity_ordered", "lines.unit_price", "lines.unit_cost", "lines.totals"},
		}),
	})
}
