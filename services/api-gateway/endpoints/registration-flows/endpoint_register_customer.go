package registrationflowep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apirequest "github.com/open-mrp/api/services/api-gateway/pkg/request"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/field"
)

// Request to register a new or existing customer.
type RegisterCustomerRequest struct {
	// Slug of the seller account the customer is registering with.
	AccountSlug string `json:"account_slug" validate:"required"`
	// Whether the registrant is an existing customer of the seller account.
	//
	// When `true`, the registering user is linked to the customer account matching `customer_number`. When `false`, a new customer account is created and `customer_name`, `customer_group_id`, `address`, `shipping_term_id`, and `payment_term_id` are required.
	IsExistingCustomer bool `json:"is_existing_customer"`
	// Customer number identifying the existing customer account.
	//
	// Required when `is_existing_customer` is `true`; ignored otherwise. New customers are assigned the seller's next customer number automatically.
	CustomerNumber field.Optional[string] `json:"customer_number,omitzero"`
	// Name for the new customer account.
	//
	// Required when registering as a new customer.
	CustomerName field.Optional[string] `json:"customer_name,omitzero"`
	// ID of the customer group to place the new customer in.
	//
	// Required when registering as a new customer.
	CustomerGroupID field.Optional[string] `json:"customer_group_id,omitzero"`
	// Contact phone number for the new customer.
	Phone field.Optional[string] `json:"phone,omitzero"`
	// Address for the new customer account.
	//
	// Required when registering as a new customer.
	Address field.Optional[apirequest.AddressInput] `json:"address,omitzero"`
	// ID of the shipping term assigned to the new customer.
	//
	// Required when registering as a new customer.
	ShippingTermID field.Optional[string] `json:"shipping_term_id,omitzero"`
	// ID of the payment term assigned to the new customer.
	//
	// Required when registering as a new customer.
	PaymentTermID field.Optional[string] `json:"payment_term_id,omitzero"`
}

var sampleRegisterStreetLine1 = "123 Main St"
var sampleRegisterLocality = "Springfield"
var sampleRegisterState = "IL"
var sampleRegisterPostalCode = "62701"

var sampleRegisterCustomerRequest = &RegisterCustomerRequest{
	AccountSlug:        "my-company",
	IsExistingCustomer: false,
	CustomerName:       field.Some("Acme Corp"),
	CustomerGroupID:    field.Some("cgrp_01abc"),
	Phone:              field.Some("+15551234567"),
	Address: field.Some(apirequest.AddressInput{
		Name:        "Headquarters",
		StreetLine1: field.SomePtr(&sampleRegisterStreetLine1),
		Locality:    field.SomePtr(&sampleRegisterLocality),
		State:       field.SomePtr(&sampleRegisterState),
		PostalCode:  field.SomePtr(&sampleRegisterPostalCode),
		Country:     "US",
	}),
	PaymentTermID:  field.Some("pt_01abc"),
	ShippingTermID: field.Some("st_01abc"),
}

func (*RegisterCustomerRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleRegisterCustomerRequest)
}

// Registers the authenticated user as a customer of a seller account.
//
// Either links the user to an existing customer account by customer number, or creates a new customer account with the provided details and links the user to it. A new customer account takes the authenticated user's email address as its contact email, and either way the seller's customer-service contacts are notified that a buyer has registered.
type RegisterCustomerEndpoint struct{}

func (e *RegisterCustomerEndpoint) Materialize() *apiendpoint.APIEndpoint[*RegisterCustomerRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*RegisterCustomerRequest, *apiresource.EmptyResource]{
		Title:             "Register Customer",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/sales/customers/registration",
		SuccessStatusCode: http.StatusCreated,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RegisterCustomerRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(RegistrationFlowSvc).RegisterCustomer
		},
	})
}
