package domain

import (
	"time"

	"github.com/open-mrp/api/shared/pagination"
)

// InventoryChangeLog represents a single inventory change log entry.
type InventoryChangeLog struct {
	ID                            string
	ItemID                        string
	ItemSKU                       string
	ItemCreatedAt                 time.Time
	ItemUpdatedAt                 time.Time
	QuantityID                    string
	QuantityValue                 string
	QuantityUnitID                string
	QuantityUnitName              string
	QuantityUnitAbbreviation      string
	QuantityUnitType              string
	QuantityUnitRatioNumerator    string
	QuantityUnitRatioDenominator  string
	QuantityUnitOffsetNumerator   string
	QuantityUnitOffsetDenominator string
	QuantityUnitCreatedAt         time.Time
	QuantityUnitUpdatedAt         time.Time
	ActionTypeCode                string
	ScanningStationID             *string
	ScanningStationName           *string
	ScanningStationType           *string
	ScanningStationCreatedAt      *time.Time
	ScanningStationUpdatedAt      *time.Time
	ItemTypeCode                  *string
	ResponsibleUserID             *string
	ResponsibleUserName           *string
	ResponsibleUserCreatedAt      *time.Time
	ResponsibleUserUpdatedAt      *time.Time
	AccountID                     string
	CreatedAt                     time.Time
	UpdatedAt                     time.Time
}

// ListInventoryChangeLogsParams contains the parameters for listing inventory change logs.
type ListInventoryChangeLogsParams struct {
	AccountID        string
	Cursor           *string
	Limit            int32
	ItemIDs          []string
	ActionTypeCodes  []string
	ChangedByUserIDs []string
	StartDate        *time.Time
	EndDate          *time.Time
}

// ListInventoryChangeLogsResult contains the paginated result of listing inventory change logs.
type ListInventoryChangeLogsResult struct {
	Items    []*InventoryChangeLog
	PageInfo pagination.PageInfo
}

// ExportInventoryChangeLogsParams contains the parameters for exporting inventory change logs.
type ExportInventoryChangeLogsParams struct {
	AccountID        string
	ItemIDs          []string
	ActionTypeCodes  []string
	ChangedByUserIDs []string
	StartDate        *time.Time
	EndDate          *time.Time
}
