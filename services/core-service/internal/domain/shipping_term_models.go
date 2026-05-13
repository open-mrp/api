package domain

import (
	"time"

	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/pagination"
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
	ID                          string
	Name                        string                     `audit:"name"`
	Type                        constants.ShippingTermType `audit:"type"`
	FlatRate                    *Quantity                  `audit:"flat_rate"`
	MinimumOrderValue           *Quantity                  `audit:"minimum_order_value"`
	FreeShippingServiceLevelIDs []string                   `audit:"free_shipping_service_level_ids"`
	FreeShippingServiceLevels   []*ServiceLevel            `audit:"-"`
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
	Includes  []string
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
	FlatRate                    *QuantityInput
	MinimumOrderValue           *QuantityInput
	FreeShippingServiceLevelIDs []string
	FlatRateID                  *string
	MinimumOrderID              *string
	Includes                    []string
}

type UpdateShippingTermParams struct {
	AccountID                      string
	ShippingTermID                 string
	Name                           *string
	Type                           *constants.ShippingTermType
	FlatRate                       *QuantityInput
	MinimumOrderValue              *QuantityInput
	FreeShippingServiceLevelIDs    []string
	HasFreeShippingServiceLevelIDs bool
	HasFlatRate                    bool
	HasMinimumOrderValue           bool
	FlatRateID                     *string
	MinimumOrderID                 *string
	Includes                       []string
}

type DeleteShippingTermParams struct {
	AccountID      string
	ShippingTermID string
}
