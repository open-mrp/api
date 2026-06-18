package domain

import (
	"time"

	"github.com/augno/api/shared/pagination"
)

type Carrier struct {
	ID                     string
	Name                   string  `audit:"name"`
	Code                   *string `audit:"code"`
	ShippoCarrierAccountID *string
	AccountNumber          *string `audit:"account_number"`
	IsPortalEnabled        bool    `audit:"is_portal_enabled"`
	AccountID              *string
	DeletedAt              *time.Time
	ServiceLevels          []*ServiceLevel
	CreatedAt              time.Time
	UpdatedAt              time.Time

	// Populated only by BatchGetCarriersByIDs when a positive service_levels_limit was requested. Not persisted; transient on the response path so the gRPC layer can mirror them into CarrierInfo's service_level_ids_preview / service_levels_has_more fields.
	ServiceLevelIDsPreview []string `audit:"-"`
	ServiceLevelsHasMore   bool     `audit:"-"`
}

type ListCarriersParams struct {
	AccountID string
	Cursor    *string
	Limit     int32
	Query     *string
	Includes  []string
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
	Code                   *string
	ShippoCarrierAccountID *string
	AccountNumber          *string
	IsPortalEnabled        bool
	ServiceLevels          []CreateServiceLevelParams
	Includes               []string
}

type UpdateCarrierParams struct {
	AccountID       string
	CarrierID       string
	Name            *string
	IsPortalEnabled *bool
	Includes        []string
}

type CreateCarrierResult struct {
	Carrier *Carrier
}
