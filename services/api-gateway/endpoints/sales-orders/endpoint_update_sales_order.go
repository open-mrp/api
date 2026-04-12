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

// UpdateSalesOrderRequest is the request to update a sales order.
type UpdateSalesOrderRequest struct {
	// The ID of the sales order to update.
	SalesOrderID string `path:"id" validate:"required"`
	// The customer purchase order number.
	CustomerPONumber *string `json:"customer_po_number,omitempty" nullable:"false" validate:"omitempty,max=255"`
	// A note for the order.
	Note *string `json:"note,omitempty" nullable:"false"`
	// The carrier ID.
	CarrierID *string `json:"carrier_id,omitempty" nullable:"true" validate:"omitempty,max=191"`
	// The service level ID.
	ServiceLevelID *string `json:"service_level_id,omitempty" nullable:"true" validate:"omitempty,max=191"`
	// The carrier billing type.
	CarrierBillingType *string `json:"carrier_billing_type,omitempty" nullable:"false" validate:"omitempty,max=255"`
	// The carrier billing account number.
	CarrierBillingAccount *string `json:"carrier_billing_account,omitempty" nullable:"false" validate:"omitempty,max=255"`
	// The priority code.
	PriorityCode *string `json:"priority_code,omitempty" nullable:"false" validate:"omitempty,max=255"`
	// The sales rep ID.
	SalesRepID *string `json:"sales_rep_id,omitempty" nullable:"true" validate:"omitempty,max=191"`
	// The shipping term ID.
	ShippingTermID *string `json:"shipping_term_id,omitempty" nullable:"true" validate:"omitempty,max=191"`
	// The payment term ID.
	PaymentTermID *string `json:"payment_term_id,omitempty" nullable:"true" validate:"omitempty,max=191"`
	// The order discount ID.
	OrderDiscountID *string `json:"order_discount_id,omitempty" nullable:"true" validate:"omitempty,max=191"`
	// Bill-to address name.
	BillToName *string `json:"bill_to_name,omitempty" nullable:"false" validate:"omitempty,max=255"`
	// Bill-to street line 1.
	BillToStreetLine1 *string `json:"bill_to_street_line_1,omitempty" nullable:"false" validate:"omitempty,max=255"`
	// Bill-to street line 2.
	BillToStreetLine2 *string `json:"bill_to_street_line_2,omitempty" nullable:"true" validate:"omitempty,max=255"`
	// Bill-to locality/city.
	BillToLocality *string `json:"bill_to_locality,omitempty" nullable:"false" validate:"omitempty,max=255"`
	// Bill-to state/province.
	BillToState *string `json:"bill_to_state,omitempty" nullable:"false" validate:"omitempty,max=255"`
	// Bill-to postal code.
	BillToPostalCode *string `json:"bill_to_postal_code,omitempty" nullable:"false" validate:"omitempty,max=255"`
	// Bill-to country.
	BillToCountry *string `json:"bill_to_country,omitempty" nullable:"false" validate:"omitempty,max=2"`
	// Ship-to address name.
	ShipToName *string `json:"ship_to_name,omitempty" nullable:"false" validate:"omitempty,max=255"`
	// Ship-to street line 1.
	ShipToStreetLine1 *string `json:"ship_to_street_line_1,omitempty" nullable:"false" validate:"omitempty,max=255"`
	// Ship-to street line 2.
	ShipToStreetLine2 *string `json:"ship_to_street_line_2,omitempty" nullable:"true" validate:"omitempty,max=255"`
	// Ship-to locality/city.
	ShipToLocality *string `json:"ship_to_locality,omitempty" nullable:"false" validate:"omitempty,max=255"`
	// Ship-to state/province.
	ShipToState *string `json:"ship_to_state,omitempty" nullable:"false" validate:"omitempty,max=255"`
	// Ship-to postal code.
	ShipToPostalCode *string `json:"ship_to_postal_code,omitempty" nullable:"false" validate:"omitempty,max=255"`
	// Ship-to country.
	ShipToCountry *string `json:"ship_to_country,omitempty" nullable:"false" validate:"omitempty,max=2"`
	// The order number.
	Number *string `json:"number,omitempty" nullable:"false" validate:"omitempty,max=255"`
	// Whether the order acknowledgment has been sent.
	IsAcknowledgmentSent *bool `json:"is_acknowledgment_sent,omitempty" nullable:"false"`
	// The promised delivery date.
	PromisedAt *time.Time `json:"promised_at,omitempty" nullable:"false"`
	// The customer (buyer account) ID.
	CustomerID *string `json:"customer_id,omitempty" nullable:"true" validate:"omitempty,max=191"`
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

type UpdateSalesOrderEndpoint struct{}

func (e *UpdateSalesOrderEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateSalesOrderRequest, *apiresource.SalesOrderDetail] {
	return &apiendpoint.APIEndpoint[*UpdateSalesOrderRequest, *apiresource.SalesOrderDetail]{
		Title:             "Update Sales Order",
		Description:       "Partially updates a sales order.",
		Method:            http.MethodPatch,
		ContentType:       "application/json",
		Route:             "/v1/sales/sales-orders/{id}",
		Request:           &UpdateSalesOrderRequest{},
		Response:          &apiresource.SalesOrderDetail{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateSalesOrderRequest) (*apiresource.SalesOrderDetail, *apierror.APIError) {
			return svc.(SalesOrderSvc).UpdateSalesOrder
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeSalesOrder,
			Fields:     []string{"customer", "bill_to_address", "ship_to_address", "carrier", "service_level", "payment_term", "shipping_term", "order_discount", "lines"},
		}),
	}
}
