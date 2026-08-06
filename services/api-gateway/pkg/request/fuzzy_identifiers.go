package apirequest

import (
	"github.com/augno/api/shared/field"
	pb "github.com/augno/api/shared/proto/core"
)

/* -------------------------- Named Object -------------------------- */
// Identifies an object by its id or its name. An id wins when both are given.
type ObjectIdentifier struct {
	// Object ID.
	ID string `json:"id,omitempty" validate:"omitempty,max=255"`
	// Object name, matched case-insensitively.
	Name string `json:"name,omitempty" validate:"omitempty,max=255"`
}

// maps a fuzzy object reference to its protobuf form.
func ObjectIdentifierToProto(d ObjectIdentifier) *pb.ObjectIdentifier {
	return &pb.ObjectIdentifier{Id: d.ID, Name: d.Name}
}

// maps an optional object reference, returning nil when unset.
func OptionalObjectIdentifierToProto(d field.Optional[ObjectIdentifier]) *pb.ObjectIdentifier {
	v, ok := d.Value()
	if !ok {
		return nil
	}
	return ObjectIdentifierToProto(v)
}

// maps a nillable object reference, returning nil when it is omitted.
func ObjectIdentifierPtrToProto(d *ObjectIdentifier) *pb.ObjectIdentifier {
	if d == nil {
		return nil
	}
	return ObjectIdentifierToProto(*d)
}

/* -------------------------- ITEM -------------------------- */
// Identifies an item by its id or its SKU. An id wins when both are given.
type ItemIdentifier struct {
	// Item ID.
	ID string `json:"id,omitempty" validate:"omitempty,max=255"`
	// Item SKU, matched case-insensitively against the account's items.
	SKU string `json:"sku,omitempty" validate:"omitempty,max=255"`
}

// maps a fuzzy item reference to its protobuf form. Empty fields cross as empty strings, which the server treats as unset.
func ItemIdentifierToProto(i ItemIdentifier) *pb.ItemIdentifier {
	return &pb.ItemIdentifier{Id: i.ID, Sku: i.SKU}
}

// maps an optional fuzzy item reference, returning nil when unset.
func OptionalItemIdentifierToProto(i field.Optional[ItemIdentifier]) *pb.ItemIdentifier {
	v, ok := i.Value()
	if !ok {
		return nil
	}
	return ItemIdentifierToProto(v)
}

/* -------------------------- UNIT -------------------------- */
// Identifies a unit by its id, its name, or its abbreviation, in that order of precedence.
type UnitIdentifier struct {
	// Unit ID.
	ID string `json:"id,omitempty" validate:"omitempty,max=255"`
	// Unit name, matched case-insensitively against the account's units.
	Name string `json:"name,omitempty" validate:"omitempty,max=255"`
	// Unit abbreviation, matched case-insensitively against the account's units.
	Abbreviation string `json:"abbreviation,omitempty" validate:"omitempty,max=255"`
}

// maps a fuzzy unit reference to its protobuf form. Empty fields cross as empty strings, which the server treats as unset.
func UnitIdentifierToProto(u UnitIdentifier) *pb.UnitIdentifier {
	return &pb.UnitIdentifier{Id: u.ID, Name: u.Name, Abbreviation: u.Abbreviation}
}

// maps an optional fuzzy unit reference, returning nil when the unit is omitted entirely.
func OptionalUnitIdentifierToProto(u field.Optional[UnitIdentifier]) *pb.UnitIdentifier {
	v, ok := u.Value()
	if !ok {
		return nil
	}
	return UnitIdentifierToProto(v)
}
