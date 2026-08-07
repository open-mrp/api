package addressvalidationep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// Request to validate an address.
type ValidateAddressRequest struct {
	// First line of the street address.
	AddressLine1 string `json:"address_line_1" validate:"required"`
	// Second line of the street address.
	AddressLine2 field.Optional[string] `json:"address_line_2,omitzero"`
	// City or locality.
	City string `json:"city" validate:"required"`
	// State or administrative area.
	State string `json:"state" validate:"required"`
	// Postal or ZIP code.
	PostalCode string `json:"postal_code" validate:"required"`
	// Two-letter country code, such as `US`.
	//
	// A full country name such as `United States` is recognized for a handful of common countries; send the two-letter code for anywhere else.
	Country string `json:"country" validate:"required"`
}

var sampleValidateAddressLine2 = "Suite 400"
var sampleValidateAddressRequest = &ValidateAddressRequest{
	AddressLine1: "123 Main St",
	AddressLine2: field.Some(sampleValidateAddressLine2),
	City:         "Springfield",
	State:        "IL",
	PostalCode:   "62701",
	Country:      "US",
}

func (*ValidateAddressRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleValidateAddressRequest)
}

// Checks an address against an address validation service and returns a standardized version of it.
//
// Nothing is created or modified. Use this before creating or updating an address to confirm it is complete and to pick up corrected values. When the service can standardize the address, `formatted_address` and `components` carry the corrected values, and `validation_messages` explains anything that was inferred, replaced, or could not be confirmed.
type ValidateAddressEndpoint struct{}

func (e *ValidateAddressEndpoint) Materialize() *apiendpoint.APIEndpoint[*ValidateAddressRequest, *apiresource.ValidatedAddress] {
	return (&apiendpoint.APIEndpoint[*ValidateAddressRequest, *apiresource.ValidatedAddress]{
		Title:             "Validate Address",
		Method:            http.MethodPut,
		ContentType:       "application/json",
		Route:             "/v1/core/addresses/actions/validate",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		ReadOnly:          true,
		Preview:           true,
		ObjectType:        constants.ObjectTypeValidatedAddress,
		Extras:            apiendpoint.APIEndpointExtras{HideFromRequestLog: true},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ValidateAddressRequest) (*apiresource.ValidatedAddress, *apierror.APIError) {
			return svc.(AddressValidationSvc).ValidateAddress
		},
	})
}
