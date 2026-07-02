package addressep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// Request to partially update an address.
//
// Omitted fields are left unchanged.
type UpdateAddressRequest struct {
	// Address ID.
	AddressID string `path:"id" validate:"required"`
	// Display name of the address.
	Name field.Optional[string] `json:"name,omitzero" validate:"omitempty,min=1,max=255"`
	// Phone number associated with the address.
	//
	// Send `null` to clear.
	Phone field.Clearable[string] `json:"phone,omitzero" validate:"omitempty,max=255"`
	// Email address associated with the address.
	//
	// Send `null` to clear.
	Email field.Clearable[string] `json:"email,omitzero" validate:"omitempty,custom_email,max=255"`
	// How the address is used.
	//
	// - `standard`: a normal shipping or billing address.
	// - `drop_ship`: an address an order is shipped to directly, typically a third party or end customer rather than the account itself.
	Type field.Optional[constants.AddressType] `json:"type,omitzero"`
	// First line of the street address.
	StreetLine1 field.Optional[string] `json:"street_line_1,omitzero" validate:"omitempty,max=255"`
	// Second line of the street address.
	//
	// Send `null` to clear.
	StreetLine2 field.Clearable[string] `json:"street_line_2,omitzero" validate:"omitempty,max=255"`
	// City or locality.
	Locality field.Optional[string] `json:"locality,omitzero" validate:"omitempty,max=255"`
	// State or administrative area.
	State field.Optional[string] `json:"state,omitzero" validate:"omitempty,max=255"`
	// Postal or ZIP code.
	PostalCode field.Optional[string] `json:"postal_code,omitzero" validate:"omitempty,max=255"`
	// Two-letter country code.
	Country field.Optional[string] `json:"country,omitzero" validate:"omitempty,max=2"`
}

var sampleUpdateName = "Warehouse"
var sampleUpdateAddressPhone = "555-123-4567"
var sampleUpdateAddressEmail = "warehouse@acme.com"
var sampleUpdateAddressStreetLine1 = "123 Main St"
var sampleUpdateAddressStreetLine2 = "Suite 400"
var sampleUpdateAddressLocality = "Springfield"
var sampleUpdateAddressState = "IL"
var sampleUpdateAddressPostalCode = "62701"
var sampleUpdateAddressCountry = "US"

var sampleUpdateAddressRequest = &UpdateAddressRequest{
	Name:        field.Some(sampleUpdateName),
	Phone:       field.Set(sampleUpdateAddressPhone),
	Email:       field.Set(sampleUpdateAddressEmail),
	Type:        field.Some(constants.AddressTypeStandard),
	StreetLine1: field.Some(sampleUpdateAddressStreetLine1),
	StreetLine2: field.Set(sampleUpdateAddressStreetLine2),
	Locality:    field.Some(sampleUpdateAddressLocality),
	State:       field.Some(sampleUpdateAddressState),
	PostalCode:  field.Some(sampleUpdateAddressPostalCode),
	Country:     field.Some(sampleUpdateAddressCountry),
}

func (*UpdateAddressRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateAddressRequest)
}

// Partially updates an address.
//
// Changing a street, locality, state, postal code, or country field may replace the address's geolocation, so the geolocation `id` in the response can change.
type UpdateAddressEndpoint struct{}

func (e *UpdateAddressEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateAddressRequest, *apiresource.Address] {
	return (&apiendpoint.APIEndpoint[*UpdateAddressRequest, *apiresource.Address]{
		Title:               "Update Address",
		Method:              http.MethodPatch,
		Route:               "/v1/sales/addresses/{id}",
		ContentType:         "application/json",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainAddresses, Action: types.ActionUpdate}, {Domain: types.PermissionDomainCustomers, Action: types.ActionUpdate}, {Domain: types.PermissionDomainSuppliers, Action: types.ActionUpdate}},
		Preview:             true,
		ObjectType:          constants.ObjectTypeAddress,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateAddressRequest) (*apiresource.Address, *apierror.APIError) {
			return svc.(AddressSvc).UpdateAddress
		},
	})
}
