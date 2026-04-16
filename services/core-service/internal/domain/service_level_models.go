package domain

import (
	"time"

	"github.com/augno/api/shared/pagination"
)

type ServiceLevel struct {
	ID                string
	Name              string  `audit:"name"`
	Code              string  `audit:"code"`
	ServiceLevelToken *string `audit:"service_level_token"`
	IsPortalEnabled   bool    `audit:"is_portal_enabled"`
	IsDefault         bool    `audit:"is_default"`
	CarrierID         string
	AccountID         *string
	CreatedAt         time.Time
	UpdatedAt         time.Time
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
	AccountID         string
	CarrierID         string
	Name              string
	Code              string
	ServiceLevelToken *string
	IsPortalEnabled   bool
	IsDefault         bool
}

type UpdateServiceLevelParams struct {
	AccountID       string
	ServiceLevelID  string
	CarrierID       string
	Name            *string
	Code            *string
	IsPortalEnabled *bool
	IsDefault       *bool
}
