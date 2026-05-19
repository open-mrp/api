package domain

import (
	"time"

	"github.com/augno/api/shared/pagination"
	"github.com/augno/api/shared/patch"
)

// Part represents a part entity (specialization of Item).
type Part struct {
	ID        string
	ItemID    string
	Item      *Item
	CreatedAt time.Time
	UpdatedAt time.Time
}

type CreatePartParams struct {
	AccountID    string
	SKU          string
	Description  *string
	Notes        *string
	CategoryID   string
	UnitPrice    *CreateRateParams
	UnitCost     *CreateRateParams
	BurnRate     *CreateRateParams
	AttributeIDs []string
	Includes     []string
}

type UpdatePartParams struct {
	AccountID   string
	PartID      string
	SKU         *string
	Description patch.Field[string]
	Notes       patch.Field[string]
	Includes    []string
}

type PartUpdateItemParams struct {
	AccountID   string
	ItemID      string
	SKU         *string
	Description patch.Field[string]
	Notes       patch.Field[string]
}

type DeletePartParams struct {
	AccountID string
	PartID    string
}

type GetPartParams struct {
	AccountID string
	PartID    string
	Includes  []string
}

type ListPartsParams struct {
	AccountID    string
	Cursor       *string
	Limit        int32
	Query        *string
	CategoryIDs  []string
	AttributeIDs []string
	StartDate    *time.Time
	EndDate      *time.Time
	Includes     []string
}

// ExportPartsParams holds filter parameters for a full (unpaginated) part export.
type ExportPartsParams struct {
	AccountID    string
	Query        *string
	CategoryIDs  []string
	AttributeIDs []string
	StartDate    *time.Time
	EndDate      *time.Time
}

type ListPartsResult struct {
	Parts    []*Part
	PageInfo pagination.PageInfo
}
