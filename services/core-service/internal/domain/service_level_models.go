package domain

import (
	"time"

	"github.com/open-mrp/apikit/field"
	"github.com/open-mrp/apikit/pagination"
)

type ServiceLevel struct {
	ID   string
	Name string `audit:"name"`
	// Code identifies the service. For a service level synced from a connected carrier it is the carrier's own code, which rates and labels are keyed on.
	Code string `audit:"code"`
	// Synced marks a service level created by syncing the carrier's services; a later sync archives it when the carrier stops offering it.
	Synced          bool `audit:"-"`
	IsPortalEnabled bool `audit:"is_portal_enabled"`
	IsActive        bool `audit:"is_active"`
	// EstimatedTransitDays is the fallback transit for this service when no lane estimate has been cached, and the only source for carriers that cannot be rated.
	EstimatedTransitDays *int32 `audit:"estimated_transit_days"`
	CarrierID            string
	AccountID            *string
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

// ServiceLevelReferences blocks deleting a service level that customers, orders, shipments or shipping terms still
// use. Being its carrier's default blocks it too, checked separately for its own message.
var ServiceLevelReferences = RegisterReferences(References{
	Noun:       "service level",
	Archivable: true,
	Probes: []ReferenceProbe{
		{Label: "customers", Table: "account_relation", Column: "default_carrier_option_id"},
		{Label: "orders", Table: "sales_order", Column: "carrier_option_id"},
		{Label: "shipments", Table: "shipment", Column: "carrier_option_id"},
		{Label: "shipping terms", Table: "shipping_term_free_shipping_rule", Column: "carrier_option_id"},
	},
})

type ListServiceLevelsParams struct {
	AccountID string
	CarrierID string
	Cursor    *string
	Limit     int32
	Query     *string
	// IsActive keeps service levels with that status; nil returns every status.
	IsActive *bool
}

type ListServiceLevelsResult struct {
	ServiceLevels []*ServiceLevel
	PageInfo      pagination.PageInfo
}

type CreateServiceLevelParams struct {
	AccountID            string
	CarrierID            string
	Name                 string
	Code                 string
	Synced               bool
	IsPortalEnabled      bool
	IsActive             bool
	EstimatedTransitDays *int32
	// MakeDefault makes the new service level its carrier's default.
	MakeDefault bool
}

type UpdateServiceLevelParams struct {
	AccountID       string
	ServiceLevelID  string
	CarrierID       string
	Name            *string
	Code            *string
	IsPortalEnabled *bool
	IsActive        *bool
	// EstimatedTransitDays is assigned rather than merged, so the service backfills the existing value when the caller leaves it unset.
	EstimatedTransitDays field.Clearable[int32]
	// MakeDefault, when set, makes the service level its carrier's default (true) or stops it being the default (false).
	MakeDefault *bool
}
