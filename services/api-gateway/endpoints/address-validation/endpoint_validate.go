package addressvalidationep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to validate an address.
type ValidateAddressRequest struct {
	// First line of the street address.
	AddressLine1 string `json:"address_line_1" validate:"required"`
	// Second line of the street address.
	AddressLine2 *string `json:"address_line_2,omitempty"`
	// City or locality.
	City string `json:"city" validate:"required"`
	// State or administrative area.
	State string `json:"state" validate:"required"`
	// Postal or ZIP code.
	PostalCode string `json:"postal_code" validate:"required"`
	// Country name or code.
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

// Validates an address and returns whether it is valid, a formatted version, and any validation messages.
type ValidateAddressEndpoint struct{}

func (e *ValidateAddressEndpoint) Materialize() *apiendpoint.APIEndpoint[*ValidateAddressRequest, *apiresource.ValidatedAddress] {
	return (&apiendpoint.APIEndpoint[*ValidateAddressRequest, *apiresource.ValidatedAddress]{
		Title:             "Validate Address",
		Method:            http.MethodPut,
		ContentType:       "application/json",
		Route:             "/v1/core/addresses/actions/validate",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ObjectType:        constants.ObjectTypeValidatedAddress,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ValidateAddressRequest) (*apiresource.ValidatedAddress, *apierror.APIError) {
			return svc.(AddressValidationSvc).ValidateAddress
		},
	})
}
