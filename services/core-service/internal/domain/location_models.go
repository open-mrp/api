package domain

import (
	"time"

	"github.com/open-mrp/api/shared/field"
	"github.com/open-mrp/api/shared/pagination"
)

// Location represents a location within an account.
// carries an export's filters — the list params without pagination, plus the cap
// that keeps one request from building an unbounded workbook
type ExportLocationsParams struct {
	AccountID string
	Query     *string
	Limit     int32
}

type Location struct {
	ID             string
	Name           string  `audit:"name"`
	TypeCode       string  `audit:"type_code"`
	ParentID       *string `audit:"parent_id"`
	ParentName     *string `audit:"parent_name"`
	ParentTypeCode *string `audit:"parent_type_code"`
	Children       []LocationChild
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// LocationChild is a lightweight child reference.
type LocationChild struct {
	ID       string
	Name     string
	TypeCode string
}

// LocationType represents a location type.
type LocationType struct {
	ID        string
	Code      string
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ListLocationsParams contains the parameters for listing locations.
type ListLocationsParams struct {
	AccountID string
	Cursor    *string
	Limit     int32
	Query     *string
	Includes  []string
}

// ListLocationsResult contains the result of listing locations.
type ListLocationsResult struct {
	Locations []*Location
	PageInfo  pagination.PageInfo
}

// GetLocationParams contains the parameters for getting a single location.
type GetLocationParams struct {
	AccountID  string
	LocationID string
	Includes   []string
}

// CreateLocationParams contains the parameters for creating a location.
type CreateLocationParams struct {
	AccountID string
	Name      string
	TypeCode  string
	ParentID  *string
	ChildIDs  []string
	Includes  []string
}

// UpdateLocationParams contains the parameters for updating a location.
type UpdateLocationParams struct {
	AccountID  string
	LocationID string
	Name       *string
	TypeCode   *string
	ParentID   field.Clearable[string]
	ChildIDs   field.Clearable[[]string]
	Includes   []string
}

// DeleteLocationParams contains the parameters for deleting a location.
type DeleteLocationParams struct {
	AccountID  string
	LocationID string
}

// ListLocationTypesParams contains the parameters for listing location types.
type ListLocationTypesParams struct {
	Cursor *string
	Limit  int32
	Query  *string
}

// GetLocationTypeParams contains the parameters for getting a single location type.
type GetLocationTypeParams struct {
	Identifier string
}

// ListLocationTypesResult contains the result of listing location types.
type ListLocationTypesResult struct {
	LocationTypes []*LocationType
	PageInfo      pagination.PageInfo
}

type UpsertLocationParams struct {
	Name     string
	TypeCode string
	// Parent references this location's parent by id or name. The parent may be another
	// row in the same batch (referenced by name).
	Parent *ObjectIdentifier
	// Children references locations to re-parent under this one, by id or name. Each may be
	// another row in the same batch (referenced by name).
	Children []ObjectIdentifier
}

// LocationRef is a resolved parent/child reference in a bulk upsert. It points either at a
// pre-existing location (ExistingID) or at another row in the same batch (BatchName — the
// lowercased row name), which the write phase resolves to that row's id once every row has
// been upserted. Exactly one field is set. No JSON tags: it round-trips job_items.
type LocationRef struct {
	ExistingID string
	BatchName  string
}

// ResolvedUpsertLocationRow is a location upsert row after resolution: its parent and child
// references resolved to either an existing location id or a same-batch row name. No JSON
// tags: the engine round-trips job_items against this type and it is an internal column.
type ResolvedUpsertLocationRow struct {
	Name     string
	TypeCode string
	Parent   *LocationRef
	Children []LocationRef
}

type BulkUpsertLocationsParams struct {
	Locations []UpsertLocationParams
}
