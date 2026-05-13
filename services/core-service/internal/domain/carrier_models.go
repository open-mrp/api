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
