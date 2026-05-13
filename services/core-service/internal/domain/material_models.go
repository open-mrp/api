package domain

import (
	"time"

	"github.com/augno/api/shared/pagination"
)

// Material represents a material entity, which extends an Item with order point and lead time quantities.
type Material struct {
	ID         string
	ItemID     string
	Item       *Item     `audit:"item"`
	OrderPoint *Quantity `audit:"order_point"`
	LeadTime   *Quantity `audit:"lead_time"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type ListMaterialsParams struct {
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

type GetMaterialParams struct {
	AccountID  string
	MaterialID string
	Includes   []string
}

// ExportMaterialsParams holds filter parameters for a full (unpaginated) material export.
type ExportMaterialsParams struct {
	AccountID    string
	Query        *string
	CategoryIDs  []string
	AttributeIDs []string
	StartDate    *time.Time
	EndDate      *time.Time
}

type ListMaterialsResult struct {
	Materials []*Material
	PageInfo  pagination.PageInfo
}

type CreateMaterialParams struct {
	AccountID       string
	ItemID          string // generated in service
	UnitValueRateID string // generated in service
	UnitCostRateID  string // generated in service
	BurnRateRateID  string // generated in service
	OrderPointID    string // generated in service
	LeadTimeID      string // generated in service
	SKU             string
	Description     *string
	Notes           *string
	CategoryID      string
	OrderPoint      *QuantityInput
	LeadTime        *QuantityInput
	UnitPrice       *CreateRateParams
	UnitCost        *CreateRateParams
	BurnRate        *CreateRateParams
	AttributeIDs    []string
	Includes        []string
}

type UpdateMaterialParams struct {
	AccountID         string
	MaterialID        string
	SKU               *string
	Description       *string
	UpdateDescription bool
	Notes             *string
	UpdateNotes       bool
	OrderPoint        *QuantityInput
	LeadTime          *QuantityInput
	UnitCost          *CreateRateParams
	Includes          []string
}

type DeleteMaterialParams struct {
	AccountID  string
	MaterialID string
}
