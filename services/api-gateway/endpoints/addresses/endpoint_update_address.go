package addressep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// UpdateAddressRequest is the request to partially update an address.
type UpdateAddressRequest struct {
	// The ID of the address to update.
	AddressID string `path:"id" validate:"required"`
	// The display name of the address.
	Name *string `json:"name,omitempty"`
	// The phone number associated with this address.
	Phone *string `json:"phone,omitempty"`
	// The email address associated with this address.
	Email *string `json:"email,omitempty"`
	// Whether this is a drop ship address.
	IsDropShip *bool `json:"is_drop_ship,omitempty"`
	// The first line of the street address.
	StreetLine1 *string `json:"street_line_1,omitempty"`
	// The second line of the street address.
	StreetLine2 *string `json:"street_line_2,omitempty"`
	// The city or locality.
	Locality *string `json:"locality,omitempty"`
	// The state or administrative area.
	State *string `json:"state,omitempty"`
	// The postal or zip code.
	PostalCode *string `json:"postal_code,omitempty"`
	// The two-letter country code.
	Country *string `json:"country,omitempty"`
}

var sampleUpdateName = "Warehouse"

var sampleUpdateAddressRequest = &UpdateAddressRequest{
	Name: &sampleUpdateName,
}

func (*UpdateAddressRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateAddressRequest)
}

type UpdateAddressEndpoint struct{}

func (e *UpdateAddressEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateAddressRequest, *apiresource.Address] {
	return &apiendpoint.APIEndpoint[*UpdateAddressRequest, *apiresource.Address]{
		Title:             "Update Address",
		Description:       "Partially updates an address.",
		Method:            http.MethodPatch,
		Route:             "/v1/sales/addresses/{id}",
		ContentType:       "application/json",
		Request:           &UpdateAddressRequest{},
		Response:          &apiresource.Address{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateAddressRequest) (*apiresource.Address, *apierror.APIError) {
			return svc.(AddressSvc).UpdateAddress
		},
	}
}
