package purchaseorderep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apirequest "github.com/open-mrp/api/services/api-gateway/pkg/request"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/field"
)

// Request to create a purchase order.
type CreatePurchaseOrderRequest struct {
	// ID of the supplier account to place the order with.
	SupplierAccountID string `json:"supplier_account_id" validate:"required"`
	// Free-form note to record on the order.
	Note field.Optional[string] `json:"note,omitzero"`
	// ID of the carrier for the order's freight.
	CarrierID field.Optional[string] `json:"carrier_id,omitzero" validate:"omitempty"`
	// ID of the carrier service level for the order's freight.
	ServiceLevelID field.Optional[string] `json:"service_level_id,omitzero" validate:"omitempty"`
	// Which party the carrier bills for freight on this order.
	//
	// - `sender`: the carrier bills the party shipping the goods.
	// - `third_party`: the carrier bills the account given in `carrier_billing_account`.
	CarrierBillingType field.Optional[string] `json:"carrier_billing_type,omitzero" validate:"omitempty,max=255"`
	// Carrier account number to bill when the billing type is `third_party`.
	CarrierBillingAccount field.Optional[string] `json:"carrier_billing_account,omitzero" validate:"omitempty,max=255"`
	// Priority level for fulfilling the order (`low`, `normal`, or `high`).
	PriorityCode string `json:"priority_code" validate:"required,max=255"`
	// ID of the shipping term that applies to the order.
	ShippingTermID field.Optional[string] `json:"shipping_term_id,omitzero" validate:"omitempty"`
	// ID of the payment term agreed with the supplier.
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
	// Bill-to country as a two-letter code.
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
	// Ship-to country as a two-letter code.
	ShipToCountry field.Optional[string] `json:"ship_to_country,omitzero" validate:"omitempty,max=2"`
	// Order lines to create with the order.
	//
	// Lines can also be added afterwards through the create-line endpoint.
	Lines []CreatePurchaseOrderLineInput `json:"lines"`
	// IDs of account users to add as email contacts on the order.
	//
	// Contacts receive the purchase order email when the order is issued with `send_email`.
	ContactAccountUserIDs []string `json:"contact_account_user_ids,omitzero"`
	// Promised delivery date in `YYYY-MM-DD` format.
	//
	// Returned as `scheduled_at` on the purchase order resource.
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
				ProductID:  apiresource.SampleProductID,
				ProductSKU: "RAW-100",
				Quantity:   apirequest.QuantityInput{Value: "500", UnitID: apiresource.SampleUnitID},
				UnitPrice:  apirequest.RateInput{Value: "12.50", NumeratorUnitID: apiresource.SampleUnitID, DenominatorUnitID: apiresource.SampleUnitID},
			},
		},
	},
}

func (*CreatePurchaseOrderRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreatePurchaseOrderRequest)
}

// Creates a purchase order.
//
// The order number is assigned automatically from a per-account sequence and the order starts in `estimate` status; issue it separately to send it to the supplier and open it for receiving. Bill-to and ship-to addresses are created as new address records from the inline address fields, and any provided lines and email contacts are created with the order.
//
// A line that references an inventory item also links that item's material to the supplier, if it is not linked already, so the material shows up as sourced from them.
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
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainPurchaseOrders, Action: types.ActionCreate},
		},
		ObjectType: constants.ObjectTypePurchaseOrder,
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
