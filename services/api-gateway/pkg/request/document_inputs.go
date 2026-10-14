package apirequest

import (
	"github.com/open-mrp/api/shared/constants"
	pb "github.com/open-mrp/api/shared/proto/core"
	apierror "github.com/open-mrp/apikit/apierror"
	apiexample "github.com/open-mrp/apikit/example"
	"github.com/open-mrp/apikit/field"
	"github.com/open-mrp/apikit/validate"
)

func init() {
	validate.RegisterWrappedTypes(field.Optional[AddressSnapshotInput]{}, field.Clearable[AddressSnapshotInput]{})
}

// The address to print on a document: a saved address by `address_id`, or the address fields inline.
//
// Either way the document keeps its own copy, so later changes to a saved address don't affect it.
type AddressSnapshotInput struct {
	// ID of a saved address to copy. Omit to give the address inline.
	AddressID field.Optional[string] `json:"address_id,omitzero"`
	// Name on the address. Required when `address_id` is omitted.
	Name field.Optional[string] `json:"name,omitzero" validate:"omitempty,min=1,max=255"`
	// First line of the street address.
	Line1 field.Optional[string] `json:"line_1,omitzero" validate:"omitempty,max=255"`
	// Second line of the street address.
	Line2 field.Optional[string] `json:"line_2,omitzero" validate:"omitempty,max=255"`
	// City or locality.
	City field.Optional[string] `json:"city,omitzero" validate:"omitempty,max=255"`
	// State or administrative area.
	State field.Optional[string] `json:"state,omitzero" validate:"omitempty,max=255"`
	// Postal or ZIP code.
	PostalCode field.Optional[string] `json:"postal_code,omitzero" validate:"omitempty,max=255"`
	// Two-letter ISO 3166-1 country code, such as `US`. Required when `address_id` is omitted.
	Country field.Optional[string] `json:"country,omitzero" validate:"omitempty,len=2"`
	// Phone number at the address.
	Phone field.Optional[string] `json:"phone,omitzero" validate:"omitempty,max=255"`
	// Email address at the address.
	Email field.Optional[string] `json:"email,omitzero" validate:"omitempty,custom_email,max=255"`
}

var sampleAddressSnapshotInput = &AddressSnapshotInput{AddressID: field.Some("ad_j8cz0b79pwdb")}

func (*AddressSnapshotInput) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleAddressSnapshotInput)
}

// Validate requires exactly one way of giving the address: a saved `address_id`, or inline fields with at least `name` and `country`. param is the field the input was sent as, such as `ship_to`.
func (in AddressSnapshotInput) Validate(param string) *apierror.APIError {
	inline := in.Name.IsSet() || in.Line1.IsSet() || in.Line2.IsSet() || in.City.IsSet() || in.State.IsSet() ||
		in.PostalCode.IsSet() || in.Country.IsSet() || in.Phone.IsSet() || in.Email.IsSet()
	switch {
	case in.AddressID.IsSet() && inline:
		return apierror.NewInvalidFormatError(param, "Give either address_id or the address fields, not both.")
	case in.AddressID.IsSet():
		return nil
	case !in.Name.IsSet() || !in.Country.IsSet():
		var fields []apierror.FieldError
		if !in.Name.IsSet() {
			fields = append(fields, apierror.Field(param+".name", apierror.CodeMissingField, "name is required for an inline address."))
		}
		if !in.Country.IsSet() {
			fields = append(fields, apierror.Field(param+".country", apierror.CodeMissingField, "country is required for an inline address."))
		}
		return apierror.NewValidationError(fields[0].Message, fields...)
	}
	return nil
}

// ToProto is the input as RPCs carry a snapshot: the saved address's ID, or the inline fields for the service to copy.
func (in AddressSnapshotInput) ToProto() *pb.AddressSnapshot {
	return &pb.AddressSnapshot{
		AddressId:  in.AddressID.Ptr(),
		Name:       ptrValue(in.Name.Ptr()),
		Line_1:     in.Line1.Ptr(),
		Line_2:     in.Line2.Ptr(),
		City:       in.City.Ptr(),
		State:      in.State.Ptr(),
		PostalCode: in.PostalCode.Ptr(),
		Country:    ptrValue(in.Country.Ptr()),
		Phone:      in.Phone.Ptr(),
		Email:      in.Email.Ptr(),
	}
}

// One person to receive documents by email, and which ones.
type EmailRecipientInput struct {
	// ID of the account user who receives the documents.
	AccountUserID string `json:"account_user_id" validate:"required"`
	// The documents they receive: `acknowledgment`, `invoice`, `shipment` or `statement`.
	Documents []constants.RecipientDocument `json:"documents" validate:"required,min=1"`
}

var sampleEmailRecipientInput = &EmailRecipientInput{
	AccountUserID: "acus_e5zu8bde0z3h",
	Documents:     []constants.RecipientDocument{constants.RecipientDocumentInvoice},
}

func (*EmailRecipientInput) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleEmailRecipientInput)
}

// EmailRecipientsToProto is a recipients list as RPCs carry it.
func EmailRecipientsToProto(in []EmailRecipientInput) []*pb.EmailRecipient {
	out := make([]*pb.EmailRecipient, len(in))
	for i, r := range in {
		docs := make([]string, len(r.Documents))
		for j, d := range r.Documents {
			docs[j] = string(d)
		}
		out[i] = &pb.EmailRecipient{AccountUserId: r.AccountUserID, Documents: docs}
	}
	return out
}

func ptrValue(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
