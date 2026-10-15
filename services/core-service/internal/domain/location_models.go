package domain

import (
	"time"

	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/apikit/field"
	"github.com/open-mrp/apikit/pagination"
)

// ExportLocationsParams carries an export's filters — the list params without pagination, plus the cap
// that keeps one request from building an unbounded workbook
type ExportLocationsParams struct {
	AccountID string
	Query     *string
	Limit     int32
}

// Location represents a location within an account.
type Location struct {
	ID             string
	Name           string                  `audit:"name"`
	TypeCode       string                  `audit:"type_code"`
	Status         constants.ArchiveStatus `audit:"status"`
	ParentID       *string                 `audit:"parent_id"`
	ParentName     *string                 `audit:"parent_name"`
	ParentTypeCode *string                 `audit:"parent_type_code"`
	// AddressID is where a site is. Only sites have one.
	AddressID *string `audit:"address_id"`
	// Children are the location's first EmbeddedLocationChildren children, newest first, when loaded.
	Children  []LocationChild
	CreatedAt time.Time
	UpdatedAt time.Time
}

// IsSite reports whether the location is a site.
func (l *Location) IsSite() bool {
	return l.TypeCode == string(constants.LocationTypeCodeSite)
}

// EmbeddedLocationChildren is how many children a loaded location carries: one more than an embedded list shows, so
// a longer list is noticed without counting it.
const EmbeddedLocationChildren = 11

// LocationChild is a lightweight child reference.
type LocationChild struct {
	ID       string
	Name     string
	TypeCode string
}

// LocationReferences blocks deleting a location that stock, movements, child locations or other records still use.
var LocationReferences = RegisterReferences(References{
	Noun:       "location",
	Archivable: true,
	Probes: []ReferenceProbe{
		{Label: "stock", Table: "inventory_receipt", Column: "storage_location_id"},
		{Label: "inventory movements", Table: "inventory_issue", Column: "storage_location_id"},
		{Label: "deliveries", Table: "delivery_line", Column: "storage_location_id"},
		{Label: "child locations", Table: "storage_location", Column: "parent_id"},
		{Label: "work centers", Table: "department", Column: "location_id"},
		{Label: "batches", Table: "batch", Column: "location_id"},
		{Label: "account settings", Table: "account", Column: "default_site_id"},
	},
})

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
	// IsActive narrows to active (true) or archived (false) locations; nil returns both.
	IsActive *bool
	// ParentIDs narrows to the direct children of these locations.
	ParentIDs []string
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
	// Status defaults to active.
	Status    *constants.ArchiveStatus
	ParentID  *string
	AddressID *string
	ChildIDs  []string
	Includes  []string
}

// UpdateLocationParams contains the parameters for updating a location.
type UpdateLocationParams struct {
	AccountID  string
	LocationID string
	Name       *string
	TypeCode   *string
	Status     *constants.ArchiveStatus
	ParentID   field.Clearable[string]
	AddressID  field.Clearable[string]
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
	// Status leaves an existing location's status as it is when nil; a new one is active.
	Status *constants.ArchiveStatus
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
	Status   *constants.ArchiveStatus
	Parent   *LocationRef
	Children []LocationRef
}

type BulkUpsertLocationsParams struct {
	Locations []UpsertLocationParams
}

// AccountSiteSeed is what an account's first site is made from.
type AccountSiteSeed struct {
	AccountName       string
	DefaultSiteID     *string
	ShippingAddressID *string
}

// LocationTreeNode is one location's place in its account's tree.
type LocationTreeNode struct {
	ID       string
	ParentID *string
	TypeCode string
}

// SiteOf is the site a location belongs to: its nearest site ancestor (itself, for a site), or defaultSiteID when no
// ancestor is a site. nodes is the account's whole tree by ID. "" when the location is unknown and there is no
// default site.
func SiteOf(nodes map[string]LocationTreeNode, locationID, defaultSiteID string) string {
	id := locationID
	// A tree deeper than the account has locations means a cycle; the bound keeps a corrupt tree from spinning.
	for range len(nodes) + 1 {
		node, ok := nodes[id]
		if !ok {
			break
		}
		if node.TypeCode == string(constants.LocationTypeCodeSite) {
			return node.ID
		}
		if node.ParentID == nil {
			break
		}
		id = *node.ParentID
	}
	return defaultSiteID
}
