package domain

import (
	"time"

	"github.com/open-mrp/api/shared/pagination"
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
	AccountID    string
	SKU          string
	Description  *string
	Notes        *string
	CategoryID   string
	OrderPoint   *QuantityInput
	LeadTime     *QuantityInput
	UnitPrice    *CreateRateParams
	UnitCost     *CreateRateParams
	AttributeIDs []string
	Includes     []string
}

// InsertMaterialItemParams is the input for writing a material's item row. It carries
// the service-generated IDs (item id + the three rate ids) alongside the item's
// caller-provided fields, so the generated IDs stay out of CreateMaterialParams.
type InsertMaterialItemParams struct {
	ItemID          string
	AccountID       string
	SKU             string
	Description     *string
	Notes           *string
	CategoryID      string
	UnitValueRateID string
	UnitCostRateID  string
	BurnRateRateID  string
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

// BulkUpsertMaterialsParams holds parameters for bulk upserting materials, matched by SKU.
type BulkUpsertMaterialsParams struct {
	Materials []UpsertMaterialParams
}

// UpsertMaterialParams is a single material to create or update in a bulk upsert. On
// create all fields apply; on update sku/description/notes/order_point/lead_time plus
// unit_price/unit_cost and properties are applied (category is create-only, matching
// the single update endpoint). Properties are additive.
type UpsertMaterialParams struct {
	SKU         string
	Description *string
	Notes       *string
	Category    ObjectIdentifier // create-only
	OrderPoint  *QuantityInput
	LeadTime    *QuantityInput
	UnitPrice   *CreateRateParams
	UnitCost    *CreateRateParams
	// Properties are resolved to attributes (find-or-create by name + value) and attached.
	Properties []UpsertItemPropertyParams
}

// MaterialSKUMatch is an existing material keyed by SKU, with the IDs needed to update it.
type MaterialSKUMatch struct {
	MaterialID      string
	ItemID          string
	SKU             string
	CategoryID      string
	UnitValueRateID string
	UnitCostRateID  string
}

// mirrors an upsert row with its category reference resolved to an id. Property
// names/values stay as written: they are found-or-created when the job runs.
type ResolvedUpsertMaterialRow struct {
	SKU         string
	Description *string
	Notes       *string
	CategoryID  string
	OrderPoint  *QuantityInput
	LeadTime    *QuantityInput
	UnitPrice   *CreateRateParams
	UnitCost    *CreateRateParams
	Properties  []UpsertItemPropertyParams
}
