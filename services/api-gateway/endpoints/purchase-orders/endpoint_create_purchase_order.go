package purchaseorderep

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

// Request to create a purchase order.
type CreatePurchaseOrderRequest struct {
	// Supplier account ID.
	SupplierAccountID string `json:"supplier_account_id" validate:"required"`
	// Order note.
	Note field.Optional[string] `json:"note,omitzero"`
	// Carrier ID.
	CarrierID field.Optional[string] `json:"carrier_id,omitzero" validate:"omitempty"`
	// Service level ID.
	ServiceLevelID field.Optional[string] `json:"service_level_id,omitzero" validate:"omitempty"`
	// Carrier billing type.
	CarrierBillingType field.Optional[string] `json:"carrier_billing_type,omitzero" validate:"omitempty,max=255"`
	// Carrier billing account number.
	CarrierBillingAccount field.Optional[string] `json:"carrier_billing_account,omitzero" validate:"omitempty,max=255"`
	// Priority code.
	PriorityCode string `json:"priority_code" validate:"required,max=255"`
	// Shipping term ID.
	ShippingTermID field.Optional[string] `json:"shipping_term_id,omitzero" validate:"omitempty"`
	// Payment term ID.
	PaymentTermID field.Optional[string] `json:"payment_term_id,omitzero" validate:"omitempty"`
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
	Lines []CreatePurchaseOrderLineInput `json:"lines"`
	// Account user IDs for email contacts.
	ContactAccountUserIDs []string `json:"contact_account_user_ids,omitzero"`
	// Promised delivery date.
	PromisedAt field.Optional[string] `json:"promised_at,omitzero"`
}

// Line item input for creating a purchase order.
type CreatePurchaseOrderLineInput struct {
	apirequest.OrderLineInput
}

var sampleCreatePONote = "Urgent restock order"
var sampleCreatePOCarrierID = apiresource.SampleCarrierID
var sampleCreatePOServiceLevelID = apiresource.SampleServiceLevelID
var sampleCreatePOShipToName = apiresource.SampleCustomerName
var sampleCreatePOShipToStreetLine1 = apiresource.SampleAddressLine1
var sampleCreatePOShipToLocality = apiresource.SampleAddressCity
var sampleCreatePOShipToState = apiresource.SampleAddressState
var sampleCreatePOShipToPostalCode = apiresource.SampleAddressPostalCode
var sampleCreatePOShipToCountry = apiresource.SampleAddressCountry
var sampleCreatePurchaseOrderRequest = &CreatePurchaseOrderRequest{
	SupplierAccountID: apiresource.SampleSupplierID,
	Note:              field.Some(sampleCreatePONote),
	CarrierID:         field.Some(sampleCreatePOCarrierID),
	ServiceLevelID:    field.Some(sampleCreatePOServiceLevelID),
	PriorityCode:      string(constants.PriorityCodeNormal),
	ShipToName:        field.Some(sampleCreatePOShipToName),
	ShipToStreetLine1: field.Some(sampleCreatePOShipToStreetLine1),
	ShipToLocality:    field.Some(sampleCreatePOShipToLocality),
	ShipToState:       field.Some(sampleCreatePOShipToState),
	ShipToPostalCode:  field.Some(sampleCreatePOShipToPostalCode),
	ShipToCountry:     field.Some(sampleCreatePOShipToCountry),
	Lines: []CreatePurchaseOrderLineInput{
		{
			OrderLineInput: apirequest.OrderLineInput{
				ProductID:                  apiresource.SampleProductID,
				ProductSKU:                 "RAW-100",
				QuantityValue:              "500",
				QuantityUnitID:             apiresource.SampleUnitID,
				UnitPriceValue:             "12.50",
				UnitPriceNumeratorUnitID:   apiresource.SampleUnitID,
				UnitPriceDenominatorUnitID: apiresource.SampleUnitID,
			},
		},
	},
}

func (*CreatePurchaseOrderRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreatePurchaseOrderRequest)
}

// Creates a purchase order.
type CreatePurchaseOrderEndpoint struct{}

func (e *CreatePurchaseOrderEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreatePurchaseOrderRequest, *apiresource.PurchaseOrder] {
	return (&apiendpoint.APIEndpoint[*CreatePurchaseOrderRequest, *apiresource.PurchaseOrder]{
		Title:             "Create Purchase Order",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/operations/purchase-orders",
		SuccessStatusCode: http.StatusCreated,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypePurchaseOrder,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreatePurchaseOrderRequest) (*apiresource.PurchaseOrder, *apierror.APIError) {
			return svc.(PurchaseOrderSvc).CreatePurchaseOrder
		},
		LocationFunc: func(resp *apiresource.PurchaseOrder) string {
			return "/v1/operations/purchase-orders/" + resp.ID
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypePurchaseOrder,
			Fields:     []string{"supplier", "bill_to_address", "ship_to_address", "freight", "payment_term", "shipping_term", "receiving_order", "lines", "contacts"},
		}),
	})
}
