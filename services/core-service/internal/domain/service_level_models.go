package domain

import (
	"time"

	"github.com/open-mrp/api/shared/field"
	"github.com/open-mrp/api/shared/pagination"
)

type ServiceLevel struct {
	ID                string
	Name              string  `audit:"name"`
	Code              string  `audit:"code"`
	ServiceLevelToken *string `audit:"service_level_token"`
	IsPortalEnabled   bool    `audit:"is_portal_enabled"`
	IsDefault         bool    `audit:"is_default"`
	// DefaultTransitDays is the fallback transit for this service when no lane estimate has been cached, and the only source for carriers that cannot be rated.
	DefaultTransitDays *int32 `audit:"default_transit_days"`
	CarrierID          string
	AccountID          *string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type ListServiceLevelsParams struct {
	AccountID string
	CarrierID string
	Cursor    *string
	Limit     int32
	Query     *string
}

type ListServiceLevelsResult struct {
	ServiceLevels []*ServiceLevel
	PageInfo      pagination.PageInfo
}

type CreateServiceLevelParams struct {
	AccountID          string
	CarrierID          string
	Name               string
	Code               string
	ServiceLevelToken  *string
	IsPortalEnabled    bool
	IsDefault          bool
	DefaultTransitDays *int32
}

type UpdateServiceLevelParams struct {
	AccountID       string
	ServiceLevelID  string
	CarrierID       string
	Name            *string
	Code            *string
	IsPortalEnabled *bool
	IsDefault       *bool
	// DefaultTransitDays is assigned rather than merged, so the service backfills the existing value when the caller leaves it unset.
	DefaultTransitDays field.Clearable[int32]
}
