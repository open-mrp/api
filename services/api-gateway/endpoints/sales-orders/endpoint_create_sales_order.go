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
)

// Request to create a sales order.
type CreateSalesOrderRequest struct {
	// Buyer account ID.
	BuyerAccountID string `json:"buyer_account_id" validate:"required,max=191"`
	// Customer purchase order number.
	CustomerPONumber *string `json:"customer_po_number,omitempty" validate:"omitempty,max=255"`
	// Order note.
	Note *string `json:"note,omitempty"`
	// Carrier ID.
	CarrierID *string `json:"carrier_id,omitempty" validate:"omitempty,max=191"`
	// Service level ID.
	ServiceLevelID *string `json:"service_level_id,omitempty" validate:"omitempty,max=191"`
	// Carrier billing type.
	CarrierBillingType *string `json:"carrier_billing_type,omitempty" validate:"omitempty,max=255"`
	// Carrier billing account number.
	CarrierBillingAccount *string `json:"carrier_billing_account,omitempty" validate:"omitempty,max=255"`
	// Priority code.
	PriorityCode string `json:"priority_code" validate:"required,max=255"`
	// Sales rep ID.
	SalesRepID *string `json:"sales_rep_id,omitempty" validate:"omitempty,max=191"`
	// Shipping term ID.
	ShippingTermID *string `json:"shipping_term_id,omitempty" validate:"omitempty,max=191"`
	// Sales order type code.
	SalesOrderTypeCode string `json:"sales_order_type_code" validate:"required,max=255"`
	// Payment term ID.
	PaymentTermID *string `json:"payment_term_id,omitempty" validate:"omitempty,max=191"`
	// Order discount ID.
	OrderDiscountID *string `json:"order_discount_id,omitempty" validate:"omitempty,max=191"`
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
	// Order lines to create.
	Lines []CreateSalesOrderLineInput `json:"lines"`
}

// Line item input for a create sales order request.
type CreateSalesOrderLineInput struct {
	apirequest.OrderLineInput
	// EDI line item ID.
	EdiLineItemID *string `json:"edi_line_item_id,omitempty" validate:"omitempty,max=191"`
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
	Note:               &sampleCreateSONote,
	CarrierID:          &sampleCreateSOCarrierID,
	ServiceLevelID:     &sampleCreateSOServiceLevelID,
	PriorityCode:       string(constants.PriorityCodeNormal),
	SalesOrderTypeCode: "standard",
	ShipToName:         &sampleCreateSOShipToName,
	ShipToStreetLine1:  &sampleCreateSOShipToStreetLine1,
	ShipToLocality:     &sampleCreateSOShipToLocality,
	ShipToState:        &sampleCreateSOShipToState,
	ShipToPostalCode:   &sampleCreateSOShipToPostalCode,
	ShipToCountry:      &sampleCreateSOShipToCountry,
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

type CreateSalesOrderEndpoint struct{}

func (e *CreateSalesOrderEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateSalesOrderRequest, *apiresource.SalesOrderDetail] {
	return &apiendpoint.APIEndpoint[*CreateSalesOrderRequest, *apiresource.SalesOrderDetail]{
		Title:             "Create Sales Order",
		Description:       "Creates a sales order.",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/sales/sales-orders",
		Request:           &CreateSalesOrderRequest{},
		Response:          &apiresource.SalesOrderDetail{},
		SuccessStatusCode: http.StatusCreated,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateSalesOrderRequest) (*apiresource.SalesOrderDetail, *apierror.APIError) {
			return svc.(SalesOrderSvc).CreateSalesOrder
		},
		LocationFunc: func(resp *apiresource.SalesOrderDetail) string {
			return "/v1/sales/sales-orders/" + resp.ID
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeSalesOrder,
			Fields:     []string{"customer", "bill_to_address", "ship_to_address", "carrier", "service_level", "payment_term", "shipping_term", "order_discount", "lines"},
		}),
	}
}
