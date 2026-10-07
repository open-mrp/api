package apirequest

import (
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/field"
	pb "github.com/open-mrp/api/shared/proto/core"
	"github.com/open-mrp/api/shared/validate"
)

func init() {
	validate.RegisterWrappedTypes(field.Optional[InlineAddressInput]{})
}

// An address saved together with the record that uses it, under that record's own permission.
//
// Without `id`, a new address is created from these fields, so `name` and `country` are required. With `id`, that saved address is updated: omitted fields are left unchanged, and `null` clears `phone`, `email`, `receive_calendar_id` or `street_line_2`. The address must already belong to the account the record saves it in.
type InlineAddressInput struct {
	// ID of a saved address to update instead of creating a new one.
	ID field.Optional[string] `json:"id,omitzero" validate:"omitempty"`
	// Display name of the address. Required when `id` is omitted.
	Name field.Optional[string] `json:"name,omitzero" validate:"omitempty,min=1,max=255"`
	// Phone number associated with the address.
	Phone field.Clearable[string] `json:"phone,omitzero" validate:"omitempty,max=255"`
	// Email address associated with the address.
	Email field.Clearable[string] `json:"email,omitzero" validate:"omitempty,custom_email,max=255"`
	// How the address is used.
	//
	// - `standard`: a normal shipping or billing address.
	// - `drop_ship`: an address an order is shipped to directly, typically a third party or end customer rather than the account itself.
	Type field.Optional[constants.AddressType] `json:"type,omitzero"`
	// The operating calendar naming the days this dock accepts freight, overriding the customer's own.
	ReceiveCalendarID field.Clearable[string] `json:"receive_calendar_id,omitzero" validate:"omitempty"`
	// First line of the street address.
	StreetLine1 field.Optional[string] `json:"street_line_1,omitzero" validate:"omitempty,max=255"`
	// Second line of the street address.
	StreetLine2 field.Clearable[string] `json:"street_line_2,omitzero" validate:"omitempty,max=255"`
	// City or locality.
	Locality field.Optional[string] `json:"locality,omitzero" validate:"omitempty,max=255"`
	// State or administrative area.
	State field.Optional[string] `json:"state,omitzero" validate:"omitempty,max=255"`
	// Postal or ZIP code.
	PostalCode field.Optional[string] `json:"postal_code,omitzero" validate:"omitempty,max=255"`
	// Two-letter ISO 3166-1 country code, such as `US`. Required when `id` is omitted.
	Country field.Optional[string] `json:"country,omitzero" validate:"omitempty,max=2"`
}

var sampleInlineAddressName = "Receiving dock"
var sampleInlineAddressStreetLine1 = "123 Main St"
var sampleInlineAddressLocality = "Springfield"
var sampleInlineAddressState = "IL"
var sampleInlineAddressPostalCode = "62701"

var sampleInlineAddressInput = &InlineAddressInput{
	Name:        field.Some(sampleInlineAddressName),
	Type:        field.Some(constants.AddressTypeStandard),
	StreetLine1: field.Some(sampleInlineAddressStreetLine1),
	Locality:    field.Some(sampleInlineAddressLocality),
	State:       field.Some(sampleInlineAddressState),
	PostalCode:  field.Some(sampleInlineAddressPostalCode),
	Country:     field.Some("US"),
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
		Phone:             field.StringClearableToProto(in.Phone),
		Email:             field.StringClearableToProto(in.Email),
		IsDropShip:        isDropShip,
		ReceiveCalendarId: field.StringClearableToProto(in.ReceiveCalendarID),
		StreetLine_1:      in.StreetLine1.Ptr(),
		StreetLine_2:      field.StringClearableToProto(in.StreetLine2),
		Locality:          in.Locality.Ptr(),
		State:             in.State.Ptr(),
		PostalCode:        in.PostalCode.Ptr(),
		Country:           in.Country.Ptr(),
	}
}
