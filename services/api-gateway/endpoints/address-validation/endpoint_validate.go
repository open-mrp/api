package addressvalidationep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// ValidateAddressRequest is the request to validate an address.
type ValidateAddressRequest struct {
	// The first line of the street address.
	AddressLine1 string `json:"address_line_1" validate:"required"`
	// The second line of the street address.
	AddressLine2 *string `json:"address_line_2,omitempty"`
	// The city.
	City string `json:"city" validate:"required"`
	// The state or administrative area.
	State string `json:"state" validate:"required"`
	// The postal or zip code.
	PostalCode string `json:"postal_code" validate:"required"`
	// The country name or code.
	Country string `json:"country" validate:"required"`
}

var sampleValidateAddressRequest = &ValidateAddressRequest{
	AddressLine1: "123 Main St",
	City:         "Springfield",
	State:        "IL",
	PostalCode:   "62701",
	Country:      "US",
}

func (*ValidateAddressRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleValidateAddressRequest)
}

type ValidateAddressEndpoint struct{}

func (e *ValidateAddressEndpoint) Materialize() *apiendpoint.APIEndpoint[*ValidateAddressRequest, *apiresource.ValidatedAddress] {
	return &apiendpoint.APIEndpoint[*ValidateAddressRequest, *apiresource.ValidatedAddress]{
		Title:             "Validate Address",
		Description:       "Validates an address and returns whether it is valid, a formatted version, and any validation messages.",
		Method:            http.MethodPost,
		Route:             "/v1/core/addresses/validate",
		Request:           &ValidateAddressRequest{},
		Response:          &apiresource.ValidatedAddress{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ValidateAddressRequest) (*apiresource.ValidatedAddress, *apierror.APIError) {
			return svc.(AddressValidationSvc).ValidateAddress
		},
	}
}
