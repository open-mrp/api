package domain

import (
	"time"

	"github.com/augno/api/shared/pagination"
)

// Location represents a location within an account.
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
	AccountID      string
	LocationID     string
	Name           *string
	TypeCode       *string
	ParentID       *string
	ChildIDs       []string
	UpdateChildren bool
	Includes       []string
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
