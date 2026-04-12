package addressep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// CreateAddressRequest is the request to create a new address.
type CreateAddressRequest struct {
	// The display name of the address.
	Name string `json:"name" validate:"required,max=255"`
	// The phone number associated with this address.
	Phone *string `json:"phone,omitempty" validate:"omitempty,max=255"`
	// The email address associated with this address.
	Email *string `json:"email,omitempty" validate:"omitnil,custom_email,max=255"`
	// Whether this is a drop ship address.
	IsDropShip bool `json:"is_drop_ship"`
	// The first line of the street address.
	StreetLine1 *string `json:"street_line_1,omitempty" validate:"omitempty,max=255"`
	// The second line of the street address.
	StreetLine2 *string `json:"street_line_2,omitempty" validate:"omitempty,max=255"`
	// The city or locality.
	Locality *string `json:"locality,omitempty" validate:"omitempty,max=255"`
	// The state or administrative area.
	State *string `json:"state,omitempty" validate:"omitempty,max=255"`
	// The postal or zip code.
	PostalCode *string `json:"postal_code,omitempty" validate:"omitempty,max=255"`
	// The two-letter country code.
	Country string `json:"country" validate:"required,max=2"`
}

var sampleCreateStreetLine1 = "123 Main St"
var sampleCreateLocality = "Springfield"
var sampleCreateState = "IL"
var sampleCreatePostalCode = "62701"

var sampleCreateAddressRequest = &CreateAddressRequest{
	Name:        "Headquarters",
	StreetLine1: &sampleCreateStreetLine1,
	Locality:    &sampleCreateLocality,
	State:       &sampleCreateState,
	PostalCode:  &sampleCreatePostalCode,
	Country:     "US",
}

func (*CreateAddressRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateAddressRequest)
}

type CreateAddressEndpoint struct{}

func (e *CreateAddressEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateAddressRequest, *apiresource.Address] {
	return &apiendpoint.APIEndpoint[*CreateAddressRequest, *apiresource.Address]{
		Title:             "Create Address",
		Description:       "Creates a new address.",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/sales/addresses",
		Request:           &CreateAddressRequest{},
		Response:          &apiresource.Address{},
		SuccessStatusCode: http.StatusCreated,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateAddressRequest) (*apiresource.Address, *apierror.APIError) {
			return svc.(AddressSvc).CreateAddress
		},
		LocationFunc: func(resp *apiresource.Address) string {
			return "/v1/sales/addresses/" + resp.ID
		},
	}
}
