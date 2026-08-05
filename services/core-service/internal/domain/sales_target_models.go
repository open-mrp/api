package domain

import "time"

// SalesTarget represents a sales target for a user.
type SalesTarget struct {
	ID           string
	StartDate    time.Time `audit:"starts_at"`
	EndDate      time.Time `audit:"ends_at"`
	SalesRepID   string    `audit:"sales_rep_id"`
	AccountID    string
	AmountID     string
	AmountValue  string `audit:"amount_value"`
	AmountUnitID string `audit:"amount_unit_id"`
	// The amount's unit, carried alongside the id so the API can render a quantity without a second lookup.
	AmountUnitName              string
	AmountUnitAbbreviation      string
	AmountUnitType              string
	AmountUnitRatioNumerator    string
	AmountUnitRatioDenominator  string
	AmountUnitOffsetNumerator   string
	AmountUnitOffsetDenominator string
	AmountUnitCreatedAt         time.Time
	AmountUnitUpdatedAt         time.Time
	CreatedAt                   time.Time
	UpdatedAt                   time.Time
}

// ListSalesTargetsParams are the parameters for listing sales targets.
type ListSalesTargetsParams struct {
	AccountID  string
	SalesRepID string
	Query      *string
	Limit      int32
	Offset     int32
}

// ListSalesTargetsResult is the result of listing sales targets.
type ListSalesTargetsResult struct {
	SalesTargets []SalesTarget
	Total        int64
}

// CreateSalesTargetParams are the parameters for creating a sales target.
type CreateSalesTargetParams struct {
	AccountID    string
	SalesRepID   string
	StartDate    time.Time
	EndDate      time.Time
	AmountValue  string
	AmountUnitID string
}

// UpsertSalesTargetParams are the parameters for upserting a sales target.
type UpsertSalesTargetParams struct {
	TargetID     string
	AccountID    string
	SalesRepID   string
	StartDate    time.Time
	EndDate      time.Time
	AmountValue  string
	AmountUnitID string
}
