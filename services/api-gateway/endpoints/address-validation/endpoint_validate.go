package addressvalidationep

import (
	"context"
	"net/http"

	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
	apiexample "github.com/open-mrp/apikit/example"
	"github.com/open-mrp/apikit/field"
)

// Request to validate an address.
type ValidateAddressRequest struct {
	// Street address.
	Line1 string `json:"line_1" validate:"required"`
	// Apartment, suite, unit or building.
	Line2 field.Optional[string] `json:"line_2,omitzero"`
	// City or locality.
	City string `json:"city" validate:"required"`
	// State, province or region; in the US and Canada, the two-letter code.
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
	Line1:      "123 Main St",
	Line2:      field.Some(sampleValidateAddressLine2),
	City:       "Springfield",
	State:      "IL",
	PostalCode: "62701",
	Country:    "US",
}

func (*ValidateAddressRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleValidateAddressRequest)
}

// Validates an address and returns a standardized version.
//
// Nothing is saved. Use it before creating or updating an address to catch missing or misspelled parts.
type ValidateAddressEndpoint struct{}

func (e *ValidateAddressEndpoint) Materialize() *apiendpoint.APIEndpoint[*ValidateAddressRequest, *apiresource.ValidatedAddress] {
	return (&apiendpoint.APIEndpoint[*ValidateAddressRequest, *apiresource.ValidatedAddress]{
		Title:             "Validate Address",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/core/addresses/actions/validate",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		ReadOnly:          true,
		ObjectType:        constants.ObjectTypeValidatedAddress,
		Extras:            apiendpoint.APIEndpointExtras{HideFromRequestLog: true},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ValidateAddressRequest) (*apiresource.ValidatedAddress, *apierror.APIError) {
			return svc.(AddressValidationSvc).ValidateAddress
		},
	})
}
