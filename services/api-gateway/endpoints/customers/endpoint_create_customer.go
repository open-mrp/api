package customerep

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

// Request to create a customer.
type CreateCustomerRequest struct {
	// Display name.
	Name string `json:"name" validate:"required,max=255"`
	// Customer number. Auto-generated if omitted.
	Number *string `json:"number,omitempty" validate:"omitempty,max=255" nullable:"false"`
	// Note.
	Note *string `json:"note,omitempty" nullable:"false"`
	// Email address.
	Email *string `json:"email,omitempty" validate:"omitempty,max=255" nullable:"false"`
	// Phone number.
	Phone *string `json:"phone,omitempty" validate:"omitempty,max=255" nullable:"false"`
	// Website URL.
	URL *string `json:"url,omitempty" validate:"omitempty,max=255" nullable:"false"`
	// Account status code.
	StatusCode *constants.AccountStatusCode `json:"status,omitempty" default:"normal" nullable:"false"`
	// EDI status.
	EDIStatus *constants.EDIStatus `json:"edi_status,omitempty" nullable:"false" default:"disabled"`
	// Commission policy.
	CommissionPolicy *constants.CommissionPolicy `json:"commission_policy,omitempty" default:"commission_exempt" nullable:"false"`
	// Freight policy.
	FreightPolicy *constants.FreightPolicy `json:"freight_policy,omitempty" default:"billed_freight" nullable:"false"`
	// Default carrier ID.
	DefaultCarrierID string `json:"default_carrier_id" validate:"required"`
	// Default service level ID.
	DefaultServiceLevelID *string `json:"default_service_level_id,omitempty" validate:"omitempty" nullable:"false"`
	// Default payment term ID.
	DefaultPaymentTermID string `json:"default_payment_term_id" validate:"required"`
	// Default shipping term ID.
	DefaultShippingTermID string `json:"default_shipping_term_id" validate:"required"`
	// Default priority code.
	DefaultPriorityCode *constants.PriorityCode `json:"default_priority,omitempty" default:"normal" nullable:"false"`
	// The ID of the account user to assign as the default sales rep.
	DefaultSalesRepID *string `json:"default_sales_rep_id,omitempty" validate:"omitempty" nullable:"false"`
	// Price group IDs.
	CustomerPriceGroupIDs []string `json:"customer_price_group_ids,omitempty" nullable:"false"`
	// Customer type group ID.
	CustomerTypeGroupID string `json:"customer_type_group_id" validate:"required"`
	// Carrier billing type.
	CarrierBillingType *constants.CarrierBillingType `json:"carrier_billing_type,omitempty" nullable:"false" default:"sender"`
	// Carrier billing account number.
	CarrierBillingAccount *string `json:"carrier_billing_account,omitempty" validate:"omitempty,max=255" nullable:"false"`
	// Credit limit.
	CreditLimit *apirequest.QuantityInput `json:"credit_limit,omitempty" nullable:"false"`
	// Bill-to address.
	BillToAddress apirequest.AddressInput `json:"bill_to_address" validate:"required"`
	// Ship-to address.
	ShipToAddress apirequest.AddressInput `json:"ship_to_address" validate:"required"`
}

var sampleCreateCustomerNote = "Key enterprise account"
var sampleCreateCustomerStreetLine1 = "123 Main St"
var sampleCreateCustomerLocality = "New York"
var sampleCreateCustomerState = "NY"
var sampleCreateCustomerPostalCode = "10001"
var sampleCreateCustomerRequest = &CreateCustomerRequest{
	Name:                  apiresource.SampleCustomerName,
	Note:                  &sampleCreateCustomerNote,
	DefaultCarrierID:      apiresource.SampleCarrierID,
	DefaultPaymentTermID:  apiresource.SamplePaymentTermID,
	DefaultShippingTermID: apiresource.SampleShippingTermID,
	CustomerTypeGroupID:   apiresource.SampleAccountGroupID,
	BillToAddress: apirequest.AddressInput{
		Name:        apiresource.SampleCustomerName,
		StreetLine1: &sampleCreateCustomerStreetLine1,
		Locality:    &sampleCreateCustomerLocality,
		State:       &sampleCreateCustomerState,
		PostalCode:  &sampleCreateCustomerPostalCode,
		Country:     "US",
	},
	ShipToAddress: apirequest.AddressInput{
		Name:        apiresource.SampleCustomerName,
		StreetLine1: &sampleCreateCustomerStreetLine1,
		Locality:    &sampleCreateCustomerLocality,
		State:       &sampleCreateCustomerState,
		PostalCode:  &sampleCreateCustomerPostalCode,
		Country:     "US",
	},
}

func (*CreateCustomerRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateCustomerRequest)
}

type CreateCustomerEndpoint struct{}

func (e *CreateCustomerEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateCustomerRequest, *apiresource.Customer] {
	return &apiendpoint.APIEndpoint[*CreateCustomerRequest, *apiresource.Customer]{
		Title:             "Create Customer",
		Description:       "Creates a customer account. Auto-generates a customer number if one is not provided.",
		Method:            http.MethodPost,
		Route:             "/v1/sales/customers",
		ContentType:       "application/json",
		Request:           &CreateCustomerRequest{},
		Response:          &apiresource.Customer{},
		SuccessStatusCode: http.StatusCreated,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateCustomerRequest) (*apiresource.Customer, *apierror.APIError) {
			return svc.(CustomerSvc).CreateCustomer
		},
		LocationFunc: func(resp *apiresource.Customer) string {
			return "/v1/sales/customers/" + resp.ID
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeCustomer,
			Fields: []string{
				"bill_to_address",
				"ship_to_address",
				"type",
				"parent_account",
				"freight_preferences.carrier",
				"freight_preferences.service_level",
				"defaults.payment_term",
				"defaults.shipping_term",
				"defaults.sales_rep",
				"defaults.priority",
				"contact_info",
				"freight_preferences",
				"defaults",
				"notification_preferences",
				"price_groups",
				"child_accounts",
				"credit_limit",
			},
		}),
	}
}
