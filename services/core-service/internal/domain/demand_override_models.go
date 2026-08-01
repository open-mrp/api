package domain

import (
	"time"

	"github.com/augno/api/shared/field"
	"github.com/augno/api/shared/pagination"
)

type DemandOverrideType struct {
	ID        string
	Code      string
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type DemandOverride struct {
	ID        string
	AccountID string

	ScopeCode  string `audit:"scope_code"`
	ScopeRefID string `audit:"scope_ref_id"`
	// ScopeName and ScopeHandle label whatever the scope points at — an item's description and SKU, or a product line's name — so a list can be rendered without a second round trip per row. Resolved on read, never stored.
	ScopeName   *string
	ScopeHandle *string

	// PeriodStartDate and PeriodEndDate bound the demand months the override applies to. They are dates rather than a single month because a merchant thinks in "Q3" or "through year end" far more often than in single months.
	PeriodStartDate time.Time `audit:"period_start_date"`
	PeriodEndDate   time.Time `audit:"period_end_date"`

	OverrideTypeCode string  `audit:"override_type_code"`
	Value            float64 `audit:"value"`
	UnitID           *string `audit:"unit_id"`

	ReasonCode *string `audit:"reason_code"`
	Note       *string `audit:"note"`

	CreatedByID string

	// EffectiveFrom and ExpiresAt bound when the override is *consulted*, which is a different axis from the period it applies to: "as of today, plan for an extra 5,000 units in Q4" stops being true once the deal is signed and real orders exist.
	EffectiveFrom time.Time  `audit:"effective_from"`
	ExpiresAt     *time.Time `audit:"expires_at"`
	IsActive      bool       `audit:"is_active"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

type ListDemandOverridesParams struct {
	AccountID         string
	Cursor            *string
	Limit             int32
	Query             *string
	ScopeCodes        []string
	ScopeRefIDs       []string
	OverrideTypeCodes []string
	IsActive          *bool
	PeriodStart       *time.Time
	PeriodEnd         *time.Time
}

type ListDemandOverridesResult struct {
	Overrides []*DemandOverride
	PageInfo  pagination.PageInfo
}

type GetDemandOverrideParams struct {
	AccountID  string
	OverrideID string
}

type CreateDemandOverrideParams struct {
	AccountID        string
	ScopeCode        string
	ScopeRefID       string
	PeriodStartDate  time.Time
	PeriodEndDate    time.Time
	OverrideTypeCode string
	Value            float64
	UnitID           *string
	ReasonCode       *string
	Note             *string
	EffectiveFrom    *time.Time
	ExpiresAt        *time.Time
	IsActive         *bool
	CreatedByID      string
}

type UpdateDemandOverrideParams struct {
	AccountID        string
	OverrideID       string
	PeriodStartDate  *time.Time
	PeriodEndDate    *time.Time
	OverrideTypeCode *string
	Value            *float64
	// The nullable columns are Clearable: unset leaves the column unchanged, clear nulls it. Clearing ExpiresAt makes the override permanent again.
	UnitID     field.Clearable[string]
	ReasonCode field.Clearable[string]
	Note       field.Clearable[string]
	ExpiresAt  field.Clearable[time.Time]
	IsActive   *bool
}

type DeleteDemandOverrideParams struct {
	AccountID  string
	OverrideID string
}
