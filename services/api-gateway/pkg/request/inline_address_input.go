package apirequest

import (
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/fieldpb"
	pb "github.com/open-mrp/api/shared/proto/core"
	apiexample "github.com/open-mrp/apikit/example"
	"github.com/open-mrp/apikit/field"
	"github.com/open-mrp/apikit/validate"
)

func init() {
	validate.RegisterWrappedTypes(field.Optional[InlineAddressInput]{})
}

// An address saved together with the record that uses it, under that record's own permission.
//
// Without `id`, a new address is created from these fields, so `name` and `country` are required. With `id`, that saved address is updated: omitted fields are left unchanged, and `null` clears `phone`, `email` or `line_2`. The address must already belong to the account the record saves it in.
type InlineAddressInput struct {
	// ID of a saved address to update instead of creating a new one.
	ID field.Optional[string] `json:"id,omitzero" validate:"omitempty"`
	// A label for the address, such as `Main warehouse`. Required when `id` is omitted.
	Name field.Optional[string] `json:"name,omitzero" validate:"omitempty,min=1,max=255"`
	// How the address is used.
	//
	// `standard`: a normal billing or shipping address. `drop_ship`: shipped to directly on the customer's behalf, such as an end customer.
	Type field.Optional[constants.AddressType] `json:"type,omitzero"`
	// Street address.
	Line1 field.Optional[string] `json:"line_1,omitzero" validate:"omitempty,max=255"`
	// Apartment, suite, unit or building.
	Line2 field.Clearable[string] `json:"line_2,omitzero" validate:"omitempty,max=255"`
	// City or locality.
	City field.Optional[string] `json:"city,omitzero" validate:"omitempty,max=255"`
	// State, province or region; in the US and Canada, the two-letter code.
	State field.Optional[string] `json:"state,omitzero" validate:"omitempty,max=255"`
	// Postal or ZIP code.
	PostalCode field.Optional[string] `json:"postal_code,omitzero" validate:"omitempty,max=255"`
	// Two-letter country code (ISO 3166-1 alpha-2), such as `US`. Required when `id` is omitted.
	Country field.Optional[string] `json:"country,omitzero" validate:"omitempty,iso3166_1_alpha2"`
	// Phone number at the address.
	Phone field.Clearable[string] `json:"phone,omitzero" validate:"omitempty,max=255"`
	// Email address at the address.
	Email field.Clearable[string] `json:"email,omitzero" validate:"omitempty,custom_email,max=255"`
	// ID of the operating calendar naming the days this location accepts freight. Send `null` to clear.
	ReceiveCalendarID field.Clearable[string] `json:"receive_calendar_id,omitzero" validate:"omitempty" visibility:"internal"`
}

var sampleInlineAddressInput = &InlineAddressInput{
	Name:       field.Some("Receiving dock"),
	Type:       field.Some(constants.AddressTypeStandard),
	Line1:      field.Some("123 Main St"),
	City:       field.Some("Springfield"),
	State:      field.Some("IL"),
	PostalCode: field.Some("62701"),
	Country:    field.Some("US"),
}

func (*InlineAddressInput) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleInlineAddressInput)
}

// InlineAddressToProto is nil when the request sent no inline address.
func InlineAddressToProto(f field.Optional[InlineAddressInput]) *pb.InlineAddressInput {
	in, ok := f.Value()
	if !ok {
		return nil
	}
	var isDropShip *bool
	if t, ok := in.Type.Value(); ok {
		v := t == constants.AddressTypeDropShip
		isDropShip = &v
	}
	return &pb.InlineAddressInput{
		Id:                in.ID.Ptr(),
		Name:              in.Name.Ptr(),
		Phone:             fieldpb.StringClearableToProto(in.Phone),
		Email:             fieldpb.StringClearableToProto(in.Email),
		IsDropShip:        isDropShip,
		ReceiveCalendarId: fieldpb.StringClearableToProto(in.ReceiveCalendarID),
		StreetLine_1:      in.Line1.Ptr(),
		StreetLine_2:      fieldpb.StringClearableToProto(in.Line2),
		Locality:          in.City.Ptr(),
		State:             in.State.Ptr(),
		PostalCode:        in.PostalCode.Ptr(),
		Country:           in.Country.Ptr(),
	}
}
