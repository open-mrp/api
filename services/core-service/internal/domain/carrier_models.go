package domain

import (
	"time"

	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/apikit/field"
	"github.com/open-mrp/apikit/pagination"
)

type Carrier struct {
	ID                     string
	Name                   string                `audit:"name"`
	Type                   constants.CarrierType `audit:"type"`
	SCAC                   *string               `audit:"scac"`
	ShippoCarrierAccountID *string
	AccountNumber          *string `audit:"account_number"`
	IsPortalEnabled        bool    `audit:"is_portal_enabled"`
	IsActive               bool    `audit:"is_active"`
	DefaultServiceLevelID  *string `audit:"default_service_level_id"`
	AccountID              *string
	ServiceLevels          []*ServiceLevel
	CreatedAt              time.Time
	UpdatedAt              time.Time

	// Populated only by BatchGetCarriersByIDs when a positive service_levels_limit was requested. Not persisted; transient on the response path so the gRPC layer can mirror them into CarrierInfo's service_level_ids_preview / service_levels_has_more fields.
	ServiceLevelIDsPreview []string `audit:"-"`
	ServiceLevelsHasMore   bool     `audit:"-"`
}

// CarrierReferences blocks deleting a carrier that customers, orders, shipments or shipping cases still use.
var CarrierReferences = RegisterReferences(References{
	Noun:       "carrier",
	Archivable: true,
	Probes: []ReferenceProbe{
		{Label: "customers", Table: "account_relation", Column: "default_carrier_id"},
		{Label: "orders", Table: "sales_order", Column: "carrier_id"},
		{Label: "shipments", Table: "shipment", Column: "carrier_id"},
		{Label: "shipping cases", Table: "shipping_case", Column: "carrier_id"},
	},
})

type ListCarriersParams struct {
	AccountID string
	Cursor    *string
	Limit     int32
	Query     *string
	// IsActive keeps carriers with that status; nil returns every status.
	IsActive *bool
	Includes []string
}

type ListCarriersResult struct {
	Carriers []*Carrier
	PageInfo pagination.PageInfo
}

type GetCarrierParams struct {
	AccountID string
	CarrierID string
	Includes  []string
}

type CreateCarrierParams struct {
	AccountID              string
	Name                   string
	Type                   constants.CarrierType
	SCAC                   *string
	ShippoCarrierAccountID *string
	AccountNumber          *string
	IsPortalEnabled        bool
	IsActive               bool
	ServiceLevels          []CreateServiceLevelParams
	Includes               []string
}

type UpdateCarrierParams struct {
	AccountID       string
	CarrierID       string
	Name            *string
	IsPortalEnabled *bool
	IsActive        *bool
	// SCAC and DefaultServiceLevelID are assigned rather than merged, so the service backfills the existing values when the caller leaves them unset.
	SCAC                  field.Clearable[string]
	DefaultServiceLevelID field.Clearable[string]
	Includes              []string
}

type CreateCarrierResult struct {
	Carrier *Carrier
}
