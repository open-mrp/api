package registrationflowep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apirequest "github.com/augno/api/services/api-gateway/pkg/request"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// Request to register a new or existing customer.
type RegisterCustomerRequest struct {
	// Account slug.
	AccountSlug string `json:"account_slug" validate:"required"`
	// Whether the registrant is an existing customer.
	IsExistingCustomer bool `json:"is_existing_customer"`
	// Customer number, if registering as an existing customer.
	CustomerNumber field.Optional[string] `json:"customer_number,omitzero"`
	// Customer name.
	CustomerName field.Optional[string] `json:"customer_name,omitzero"`
	// Customer group ID.
	CustomerGroupID field.Optional[string] `json:"customer_group_id,omitzero"`
	// Phone number.
	Phone field.Optional[string] `json:"phone,omitzero"`
	// Customer address.
	Address field.Optional[apirequest.AddressInput] `json:"address,omitzero"`
	// Shipping term ID.
	ShippingTermID field.Optional[string] `json:"shipping_term_id,omitzero"`
	// Payment term ID.
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

// Registers a customer through a registration flow.
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
