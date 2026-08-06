package domain

import (
	"time"

	"github.com/augno/api/shared/field"
	"github.com/augno/api/shared/pagination"
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
	AttributeIDs []string
	Includes     []string
}

type UpdatePartParams struct {
	AccountID   string
	PartID      string
	SKU         *string
	Description field.Clearable[string]
	Notes       field.Clearable[string]
	Includes    []string
}

type PartUpdateItemParams struct {
	AccountID   string
	ItemID      string
	SKU         *string
	Description field.Clearable[string]
	Notes       field.Clearable[string]
}

type DeletePartParams struct {
	AccountID string
	PartID    string
}

// BulkUpsertPartsParams holds parameters for bulk upserting parts, matched by SKU.
type BulkUpsertPartsParams struct {
	Parts []UpsertPartParams
}

// UpsertPartParams is a single part to create or update in a bulk upsert. On create
// all fields apply; on update sku/description/notes/unit_price/unit_cost/properties
// are applied (properties are additive).
type UpsertPartParams struct {
	SKU         string
	Description *string
	Notes       *string
	Category    ObjectIdentifier // create-only
	UnitPrice   *CreateRateParams
	UnitCost    *CreateRateParams
	// Properties are resolved to attributes (find-or-create by name + value) and attached.
	Properties []UpsertItemPropertyParams
}

// PartSKUMatch is an existing part keyed by SKU, with the IDs needed to update it.
type PartSKUMatch struct {
	PartID          string
	ItemID          string
	SKU             string
	CategoryID      string
	UnitValueRateID string
	UnitCostRateID  string
}

// mirrors an upsert row with its category reference resolved to an id. Property
// names/values stay as written: they are found-or-created when the job runs.
type ResolvedUpsertPartRow struct {
	SKU         string
	Description *string
	Notes       *string
	CategoryID  string
	UnitPrice   *CreateRateParams
	UnitCost    *CreateRateParams
	Properties  []UpsertItemPropertyParams
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
