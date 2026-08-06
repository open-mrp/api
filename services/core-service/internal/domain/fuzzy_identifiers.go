package domain

// Fuzzy reference types: a request points at an existing entity by any one of several
// identifiers, and the service resolves it. An empty field means unset; when more than
// one is set the documented precedence applies.

// UnitIdentifier identifies a unit by its ID, name, or abbreviation. Precedence: ID, then
// name, then abbreviation. The zero value means no unit was provided.
type UnitIdentifier struct {
	ID           string
	Name         string
	Abbreviation string
}

// ItemIdentifier identifies an item by its ID or SKU. Precedence: ID, then SKU. The zero
// value means no item was provided.
type ItemIdentifier struct {
	ID  string
	SKU string
}

// ObjectIdentifier identifies an object by its ID or name. Precedence: ID, then name.
type ObjectIdentifier struct {
	ID   string
	Name string
}
