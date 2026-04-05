package domain

import "github.com/shopspring/decimal"

// OpenBatchSummary represents an aggregated open batch summary for analytics.
type OpenBatchSummary struct {
	DepartmentName    string
	ItemName          string
	ItemID            string
	ScanningStationID string
	Count             decimal.Decimal
	Unit              string
}
