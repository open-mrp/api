package domain

import (
	"time"

	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/apikit/field"
	"github.com/open-mrp/apikit/pagination"
)

type Quantity struct {
	ID               string
	Value            string `audit:"value"`
	UnitID           string `audit:"unit_id"`
	UnitName         string `audit:"unit_name"`
	UnitAbbreviation string `audit:"unit_abbreviation"`
	UnitType         string `audit:"unit_type"`
	CreatedAt        time.Time
	UpdatedAt        time.Time
	EmbeddedUnit     *Unit `audit:"-"`
}

type ShippingTerm struct {
	ID                string
	Name              string                     `audit:"name"`
	Type              constants.ShippingTermType `audit:"type"`
	Status            constants.ArchiveStatus    `audit:"status"`
	FlatRate          *Quantity                  `audit:"flat_rate"`
	MinimumOrderValue *Quantity                  `audit:"minimum_order_value"`
	// FlatRateAmount and MinimumOrderAmount are FlatRate and MinimumOrderValue as money.
	FlatRateAmount              *Money          `audit:"-"`
	MinimumOrderAmount          *Money          `audit:"-"`
	FreeShippingServiceLevelIDs []string        `audit:"free_shipping_service_level_ids"`
	FreeShippingServiceLevels   []*ServiceLevel `audit:"-"`
	AccountID                   *string
	CreatedAt                   time.Time
	UpdatedAt                   time.Time
}

type QuantityInput struct {
	Value  string
	UnitID string
}

type ListShippingTermsParams struct {
	AccountID string
	Cursor    *string
	Limit     int32
	Query     *string
	// IsActive narrows to active (true) or archived (false) terms; nil returns both.
	IsActive *bool
	Includes []string
}

type ListShippingTermsResult struct {
	ShippingTerms []*ShippingTerm
	PageInfo      pagination.PageInfo
}

type GetShippingTermParams struct {
	AccountID      string
	ShippingTermID string
	Includes       []string
}

type CreateShippingTermParams struct {
	AccountID                   string
	Name                        string
	Type                        constants.ShippingTermType
	Status                      constants.ArchiveStatus
	FlatRate                    *QuantityInput
	MinimumOrderValue           *QuantityInput
	FreeShippingServiceLevelIDs []string
	FlatRateID                  *string
	MinimumOrderID              *string
	Includes                    []string
}

type UpdateShippingTermParams struct {
	AccountID                   string
	ShippingTermID              string
	Name                        *string
	Type                        *constants.ShippingTermType
	Status                      *constants.ArchiveStatus
	FlatRate                    field.Clearable[QuantityInput]
	MinimumOrderValue           field.Clearable[QuantityInput]
	FreeShippingServiceLevelIDs field.Clearable[[]string]
	FlatRateID                  *string
	MinimumOrderID              *string
	Includes                    []string
}

type DeleteShippingTermParams struct {
	AccountID      string
	ShippingTermID string
}

// ShippingTermReferences blocks deleting a shipping term a customer or order still uses.
var ShippingTermReferences = RegisterReferences(References{
	Noun:       "shipping term",
	Archivable: true,
	Probes: []ReferenceProbe{
		{Label: "customers", Table: "account_relation", Column: "shipping_term_id"},
		{Label: "sales orders", Table: "sales_order", Column: "shipping_term_id"},
	},
})
