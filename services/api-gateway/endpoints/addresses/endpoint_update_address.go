package addressep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
	apiexample "github.com/open-mrp/apikit/example"
	"github.com/open-mrp/apikit/field"
)

// Request to update an address. Omitted fields are left unchanged.
type UpdateAddressRequest struct {
	// Address ID.
	AddressID string `path:"id" validate:"required"`
	// A label for the address, such as `Main warehouse`.
	Name field.Optional[string] `json:"name,omitzero" validate:"omitempty,min=1,max=255"`
	// How the address is used.
	//
	// `standard`: a normal billing or shipping address. `drop_ship`: shipped to directly on the customer's behalf, such as an end customer.
	Type field.Optional[constants.AddressType] `json:"type,omitzero"`
	// Street address.
	Line1 field.Optional[string] `json:"line_1,omitzero" validate:"omitempty,max=255"`
	// Apartment, suite, unit or building. Send `null` to clear.
	Line2 field.Clearable[string] `json:"line_2,omitzero" validate:"omitempty,max=255"`
	// City or locality.
	City field.Optional[string] `json:"city,omitzero" validate:"omitempty,max=255"`
	// State, province or region; in the US and Canada, the two-letter code.
	State field.Optional[string] `json:"state,omitzero" validate:"omitempty,max=255"`
	// Postal or ZIP code.
	PostalCode field.Optional[string] `json:"postal_code,omitzero" validate:"omitempty,max=255"`
	// Two-letter country code (ISO 3166-1 alpha-2), such as `US`.
	Country field.Optional[string] `json:"country,omitzero" validate:"omitempty,iso3166_1_alpha2"`
	// Phone number at the address. Send `null` to clear.
	Phone field.Clearable[string] `json:"phone,omitzero" validate:"omitempty,max=255"`
	// Email address at the address. Send `null` to clear.
	Email field.Clearable[string] `json:"email,omitzero" validate:"omitempty,custom_email,max=255"`
	// ID of the operating calendar naming the days this location accepts freight. Send `null` to use the customer's calendar.
	ReceiveCalendarID field.Clearable[string] `json:"receive_calendar_id,omitzero" validate:"omitempty" visibility:"internal"`
}

var sampleUpdateAddressRequest = &UpdateAddressRequest{
	Name:       field.Some("Warehouse"),
	Type:       field.Some(constants.AddressTypeStandard),
	Line1:      field.Some("123 Main St"),
	Line2:      field.Set("Suite 400"),
	City:       field.Some("Springfield"),
	State:      field.Some("IL"),
	PostalCode: field.Some("62701"),
	Country:    field.Some("US"),
	Phone:      field.Set("555-123-4567"),
	Email:      field.Set("warehouse@acme.com"),
}

func (*UpdateAddressRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateAddressRequest)
}

// Updates an address.
//
// Only fields you send are changed. Documents already created keep the address as it was when they were created.
type UpdateAddressEndpoint struct{}

func (e *UpdateAddressEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateAddressRequest, *apiresource.Address] {
	return (&apiendpoint.APIEndpoint[*UpdateAddressRequest, *apiresource.Address]{
		Title:             "Update Address",
		Method:            http.MethodPatch,
		Route:             addressRoute,
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainAddresses, Action: types.ActionUpdate}}, Counterparty: authz.Counterparties(types.ActionUpdate)},
		ObjectType:        constants.ObjectTypeAddress,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateAddressRequest) (*apiresource.Address, *apierror.APIError) {
			return svc.(AddressSvc).UpdateAddress
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAddress,
			Fields:     []string{"receive_calendar"},
		}),
	})
}
