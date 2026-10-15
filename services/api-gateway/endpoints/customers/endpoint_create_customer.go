package customerep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apirequest "github.com/open-mrp/api/services/api-gateway/pkg/request"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
	apiexample "github.com/open-mrp/apikit/example"
	"github.com/open-mrp/apikit/field"
)

// Request to create a customer.
type CreateCustomerRequest struct {
	// The customer's business name, as shown throughout the app and on documents.
	Name string `json:"name" validate:"required,max=255"`
	// Customer number, unique within your account. If omitted, the next number in your sequence is assigned.
	Number field.Optional[string] `json:"number,omitzero" validate:"omitempty,max=255"`
	// Whether the customer can receive new orders: `active` or `archived`. Defaults to `active`.
	Status field.Optional[constants.ArchiveStatus] `json:"status,omitzero"`
	// The customer's credit standing: `normal`, `hold_shipment` or `hold_all`. Defaults to `normal`.
	Standing field.Optional[constants.CustomerStanding] `json:"standing,omitzero"`
	// Your team's standing instructions for this customer's orders.
	Instructions field.Optional[string] `json:"instructions,omitzero"`
	// Whether orders from this customer earn sales commission: `applied` or `exempt`. Defaults to `exempt`.
	CommissionPolicy field.Optional[constants.CommissionPolicy] `json:"commission_policy,omitzero"`
	// Whether this customer is billed for freight: `billed` or `free`. Defaults to `billed`.
	FreightPolicy field.Optional[constants.FreightPolicy] `json:"freight_policy,omitzero"`
	// Credit extended to this customer. Omit for no limit.
	CreditLimit field.Optional[apirequest.MoneyInput] `json:"credit_limit,omitzero"`
	// How to reach the customer's business.
	Contact field.Optional[CustomerContactInput] `json:"contact,omitzero"`
	// Who pays the carrier.
	CarrierBilling field.Optional[CustomerCarrierBillingInput] `json:"carrier_billing,omitzero"`
	// Values copied onto a new order for this customer when the order doesn't set them.
	Defaults CreateCustomerDefaultsInput `json:"defaults" validate:"required"`
	// ID of the account group that classifies this customer.
	GroupID string `json:"group_id" validate:"required"`
	// IDs of the pricing account groups the customer belongs to.
	PriceGroupIDs []string `json:"price_group_ids,omitzero" visibility:"internal"`
	// ID of the customer this one belongs to, making it a child customer.
	ParentCustomerID field.Optional[string] `json:"parent_customer_id,omitzero"`
	// The customer's first billing address. It becomes `defaults.bill_to_address`.
	BillToAddress apirequest.AddressInput `json:"bill_to_address" validate:"required"`
	// The customer's first shipping address. It becomes `defaults.ship_to_address`; one identical to the billing address is saved once.
	ShipToAddress apirequest.AddressInput `json:"ship_to_address" validate:"required"`
}

var sampleCreateCustomerLine1 = "123 Main St"
var sampleCreateCustomerCity = "New York"
var sampleCreateCustomerState = "NY"
var sampleCreateCustomerPostalCode = "10001"
var sampleCreateCustomerRequest = &CreateCustomerRequest{
	Name:             apiresource.SampleCustomerName,
	Number:           field.Some(apiresource.SampleCustomerNumber),
	Standing:         field.Some(constants.CustomerStandingNormal),
	Instructions:     field.Some("Send invoices to accounts payable."),
	CommissionPolicy: field.Some(constants.CommissionPolicyApplied),
	FreightPolicy:    field.Some(constants.FreightPolicyBilled),
	CreditLimit:      field.Some(apirequest.MoneyInput{Amount: "10000.00", Currency: "usd"}),
	Contact: field.Some(CustomerContactInput{
		Email: field.Set("orders@acme.com"),
		Phone: field.Set("555-123-4567"),
		URL:   field.Set("https://acme.com"),
	}),
	CarrierBilling: field.Some(CustomerCarrierBillingInput{Type: field.Some(constants.CarrierBillingTypeSender)}),
	Defaults: CreateCustomerDefaultsInput{
		CarrierID:      apiresource.SampleCarrierID,
		ServiceLevelID: field.Some(apiresource.SampleServiceLevelID),
		PaymentTermID:  apiresource.SamplePaymentTermID,
		ShippingTermID: apiresource.SampleShippingTermID,
		SalesRepID:     field.Some(apiresource.SampleAccountUserID),
		LeadTime:       field.Some(apirequest.QuantityInput{Value: "10", UnitID: apirequest.DayUnitID}),
		Supply:         field.Some(constants.SupplyStock),
	},
	GroupID: apiresource.SampleAccountGroupID,
	BillToAddress: apirequest.AddressInput{
		Name:       apiresource.SampleCustomerName,
		Line1:      field.SomePtr(&sampleCreateCustomerLine1),
		City:       field.SomePtr(&sampleCreateCustomerCity),
		State:      field.SomePtr(&sampleCreateCustomerState),
		PostalCode: field.SomePtr(&sampleCreateCustomerPostalCode),
		Country:    "US",
	},
	ShipToAddress: apirequest.AddressInput{
		Name:       apiresource.SampleCustomerName,
		Line1:      field.SomePtr(&sampleCreateCustomerLine1),
		City:       field.SomePtr(&sampleCreateCustomerCity),
		State:      field.SomePtr(&sampleCreateCustomerState),
		PostalCode: field.SomePtr(&sampleCreateCustomerPostalCode),
		Country:    "US",
	},
}

func (*CreateCustomerRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateCustomerRequest)
}

// Creates a customer.
//
// Give its first billing and shipping addresses inline. They become the customer's default addresses. If `number` is omitted, the next number in your sequence is assigned.
type CreateCustomerEndpoint struct{}

func (e *CreateCustomerEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateCustomerRequest, *apiresource.Customer] {
	return (&apiendpoint.APIEndpoint[*CreateCustomerRequest, *apiresource.Customer]{
		Title:             "Create Customer",
		Method:            http.MethodPost,
		Route:             "/v1/sales/customers",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusCreated,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainCustomers, Action: types.ActionCreate}}},
		ObjectType:        constants.ObjectTypeCustomer,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateCustomerRequest) (*apiresource.Customer, *apierror.APIError) {
			return svc.(CustomerSvc).CreateCustomer
		},
		LocationFunc: func(resp *apiresource.Customer) string {
			return "/v1/sales/customers/" + resp.ID
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeCustomer,
			Fields:     customerIncludeFields,
		}),
	})
}
