package addressep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to partially update an address.
type UpdateAddressRequest struct {
	// Address ID.
	AddressID string `path:"id" validate:"required"`
	// Display name of the address.
	Name *string `json:"name,omitempty" nullable:"false" validate:"omitempty,max=255"`
	// Phone number associated with the address.
	Phone *string `json:"phone,omitempty" nullable:"true" validate:"omitempty,max=255"`
	// Email address associated with the address.
	Email *string `json:"email,omitempty" nullable:"true" validate:"omitempty,max=255"`
	// Whether the address is a drop ship location.
	IsDropShip *bool `json:"is_drop_ship,omitempty" nullable:"false"`
	// First line of the street address.
	StreetLine1 *string `json:"street_line_1,omitempty" nullable:"false" validate:"omitempty,max=255"`
	// Second line of the street address.
	StreetLine2 *string `json:"street_line_2,omitempty" nullable:"true" validate:"omitempty,max=255"`
	// City or locality.
	Locality *string `json:"locality,omitempty" nullable:"false" validate:"omitempty,max=255"`
	// State or administrative area.
	State *string `json:"state,omitempty" nullable:"false" validate:"omitempty,max=255"`
	// Postal or ZIP code.
	PostalCode *string `json:"postal_code,omitempty" nullable:"false" validate:"omitempty,max=255"`
	// Two-letter country code.
	Country *string `json:"country,omitempty" nullable:"false" validate:"omitempty,max=2"`
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
