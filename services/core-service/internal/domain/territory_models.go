package domain

import (
	"time"

	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/pagination"
)

// TerritorySalesRep represents the sales rep sub-resource within a territory.
type TerritorySalesRep struct {
	ID        string
	Name      *string
	Email     *string
	Status    *constants.AccountUserStatus
	CreatedAt *time.Time
	UpdatedAt *time.Time
}

// TerritoryProductLine represents the product line sub-resource within a territory.
type TerritoryProductLine struct {
	ID               string
	Name             string
	CommissionPolicy *constants.CommissionPolicy
	FreightPolicy    *constants.FreightPolicy
	CreatedAt        *time.Time
	UpdatedAt        *time.Time
}

// Territory represents a sales rep territory assignment.
type Territory struct {
	ID           string
	State        string `audit:"state"`
	StartZipcode *int32 `audit:"start_zipcode"`
	EndZipcode   *int32 `audit:"end_zipcode"`
	SalesRepID   string
	SalesRep     *TerritorySalesRep    `audit:"sales_rep"`
	ProductLine  *TerritoryProductLine `audit:"product_line"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// ListTerritoriesParams contains the parameters for listing territories.
type ListTerritoriesParams struct {
	AccountID string
	Cursor    *string
	Limit     int32
	Query     *string
	Includes  []string
}

// ListTerritoriesResult contains the result of listing territories.
type ListTerritoriesResult struct {
	Territories []*Territory
	PageInfo    pagination.PageInfo
}

// GetTerritoryParams contains the parameters for getting a territory.
type GetTerritoryParams struct {
	AccountID   string
	TerritoryID string
	Includes    []string
}

// CreateTerritoryParams contains the parameters for creating a territory.
type CreateTerritoryParams struct {
	AccountID     string
	State         string
	StartZipcode  *int32
	EndZipcode    *int32
	SalesRepID    string
	ProductLineID *string
	Includes      []string
}

// UpdateTerritoryParams contains the parameters for updating a territory.
type UpdateTerritoryParams struct {
	AccountID         string
	TerritoryID       string
	State             *string
	StartZipcode      *int32
	EndZipcode        *int32
	SalesRepID        *string
	ProductLineID     *string
	ClearProductLine  bool
	ClearStartZipcode bool
	ClearEndZipcode   bool
	Includes          []string
}

// DeleteTerritoryParams contains the parameters for deleting a territory.
type DeleteTerritoryParams struct {
	AccountID   string
	TerritoryID string
}
