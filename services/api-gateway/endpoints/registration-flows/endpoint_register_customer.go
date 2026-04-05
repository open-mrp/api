package registrationflowep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// RegisterCustomerAddressRequest is the address information for a customer registration.
type RegisterCustomerAddressRequest struct {
	// The first line of the street address.
	StreetLine1 string `json:"street_line_1" validate:"required"`
	// The second line of the street address.
	StreetLine2 *string `json:"street_line_2,omitempty"`
	// The city or locality.
	Locality string `json:"locality" validate:"required"`
	// The state or region.
	State string `json:"state" validate:"required"`
	// The postal code.
	PostalCode string `json:"postal_code" validate:"required"`
	// The country code.
	Country string `json:"country" validate:"required"`
	// The name associated with the address.
	Name *string `json:"name,omitempty"`
}

// RegisterCustomerRequest is the request to register a new or existing customer.
type RegisterCustomerRequest struct {
	// The slug of the account to register with.
	AccountSlug string `json:"account_slug" validate:"required"`
	// Whether the registrant is an existing customer.
	IsExistingCustomer bool `json:"is_existing_customer"`
	// The customer number, if registering as an existing customer.
	CustomerNumber *string `json:"customer_number,omitempty"`
	// The name of the customer.
	CustomerName *string `json:"customer_name,omitempty"`
	// The ID of the customer group to associate with the customer.
	CustomerGroupID *string `json:"customer_group_id,omitempty"`
	// The phone number of the customer.
	Phone *string `json:"phone,omitempty"`
	// The address of the customer.
	Address *RegisterCustomerAddressRequest `json:"address,omitempty"`
	// The ID of the shipping term to associate with the customer.
	ShippingTermID *string `json:"shipping_term_id,omitempty"`
	// The ID of the payment term to associate with the customer.
	PaymentTermID *string `json:"payment_term_id,omitempty"`
}

var sampleRegisterCustomerRequest = &RegisterCustomerRequest{
	AccountSlug:        "my-company",
	IsExistingCustomer: false,
	CustomerName:       new("Acme Corp"),
	CustomerGroupID:    new("cgrp_01abc"),
	Phone:              new("+15551234567"),
	Address: &RegisterCustomerAddressRequest{
		StreetLine1: "123 Main St",
		Locality:    "Springfield",
		State:       "IL",
		PostalCode:  "62701",
		Country:     "US",
	},
	PaymentTermID:  new("pt_01abc"),
	ShippingTermID: new("st_01abc"),
}

func (*RegisterCustomerRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleRegisterCustomerRequest)
}

type RegisterCustomerEndpoint struct{}

func (e *RegisterCustomerEndpoint) Materialize() *apiendpoint.APIEndpoint[*RegisterCustomerRequest, *apiresource.EmptyResource] {
	return &apiendpoint.APIEndpoint[*RegisterCustomerRequest, *apiresource.EmptyResource]{
		Title:             "Register Customer",
		Description:       "Submits a customer registration request through a registration flow.",
		Method:            http.MethodPost,
		Route:             "/v1/sales/customers/registration",
		Request:           &RegisterCustomerRequest{},
		Response:          &apiresource.EmptyResource{},
		SuccessStatusCode: http.StatusCreated,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RegisterCustomerRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(RegistrationFlowSvc).RegisterCustomer
		},
	}
}
