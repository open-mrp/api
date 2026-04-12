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
)

// CreatePurchaseOrderRequest is the request to create a new purchase order.
type CreatePurchaseOrderRequest struct {
	// The supplier account ID.
	SupplierAccountID string `json:"supplier_account_id" validate:"required,max=191"`
	// A note for the order.
	Note *string `json:"note,omitempty"`
	// The carrier ID.
	CarrierID *string `json:"carrier_id,omitempty" validate:"omitempty,max=191"`
	// The service level ID.
	ServiceLevelID *string `json:"service_level_id,omitempty" validate:"omitempty,max=191"`
	// The carrier billing type.
	CarrierBillingType *string `json:"carrier_billing_type,omitempty" validate:"omitempty,max=255"`
	// The carrier billing account number.
	CarrierBillingAccount *string `json:"carrier_billing_account,omitempty" validate:"omitempty,max=255"`
	// The priority code.
	PriorityCode string `json:"priority_code" validate:"required,max=255"`
	// The shipping term ID.
	ShippingTermID *string `json:"shipping_term_id,omitempty" validate:"omitempty,max=191"`
	// The payment term ID.
	PaymentTermID *string `json:"payment_term_id,omitempty" validate:"omitempty,max=191"`
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
	// The order lines to create.
	Lines []CreatePurchaseOrderLineInput `json:"lines"`
	// The account user IDs for email contacts.
	ContactAccountUserIDs []string `json:"contact_account_user_ids,omitempty"`
	// The promised/scheduled delivery date.
	PromisedAt *string `json:"promised_at,omitempty"`
}

// CreatePurchaseOrderLineInput represents a line item in a create purchase order request.
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
	Note:              &sampleCreatePONote,
	CarrierID:         &sampleCreatePOCarrierID,
	ServiceLevelID:    &sampleCreatePOServiceLevelID,
	PriorityCode:      string(constants.PriorityCodeNormal),
	ShipToName:        &sampleCreatePOShipToName,
	ShipToStreetLine1: &sampleCreatePOShipToStreetLine1,
	ShipToLocality:    &sampleCreatePOShipToLocality,
	ShipToState:       &sampleCreatePOShipToState,
	ShipToPostalCode:  &sampleCreatePOShipToPostalCode,
	ShipToCountry:     &sampleCreatePOShipToCountry,
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

type CreatePurchaseOrderEndpoint struct{}

func (e *CreatePurchaseOrderEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreatePurchaseOrderRequest, *apiresource.PurchaseOrderDetail] {
	return &apiendpoint.APIEndpoint[*CreatePurchaseOrderRequest, *apiresource.PurchaseOrderDetail]{
		Title:             "Create Purchase Order",
		Description:       "Creates a new purchase order.",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/operations/purchase-orders",
		Request:           &CreatePurchaseOrderRequest{},
		Response:          &apiresource.PurchaseOrderDetail{},
		SuccessStatusCode: http.StatusCreated,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreatePurchaseOrderRequest) (*apiresource.PurchaseOrderDetail, *apierror.APIError) {
			return svc.(PurchaseOrderSvc).CreatePurchaseOrder
		},
		LocationFunc: func(resp *apiresource.PurchaseOrderDetail) string {
			return "/v1/operations/purchase-orders/" + resp.ID
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypePurchaseOrder,
			Fields:     []string{"supplier", "bill_to_address", "ship_to_address", "carrier", "service_level", "payment_term", "shipping_term", "receiving_order", "lines", "contacts"},
		}),
	}
}
