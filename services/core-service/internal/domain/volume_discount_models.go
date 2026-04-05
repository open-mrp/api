package domain

import (
	"time"

	"github.com/augno/api/shared/pagination"
)

type VolumeDiscount struct {
	ID              string
	Name            string `audit:"name"`
	AccountID       string
	Tiers           []*VolumeDiscountTier          `audit:"tiers"`
	CustomerGroups  []*VolumeDiscountCustomerGroup `audit:"customer_groups"`
	ProductLines    []*VolumeDiscountProductLine   `audit:"product_lines"`
	Categories      []*VolumeDiscountCategory      `audit:"categories"`
	Attributes      []*VolumeDiscountAttribute     `audit:"attributes"`
	AcceptableUnits []*VolumeDiscountUnit          `audit:"acceptable_units"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type VolumeDiscountTier struct {
	ID                 string
	Name               string
	DiscountPercentage string
	Threshold          string
	ParentTierID       *string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type VolumeDiscountCustomerGroup struct {
	ID             string
	AccountGroupID string
	Name           string
}

type VolumeDiscountProductLine struct {
	ID   string
	Name string
}

type VolumeDiscountCategory struct {
	ID   string
	Name string
}

type VolumeDiscountAttribute struct {
	ID   string
	Name string
}

type VolumeDiscountUnit struct {
	ID           string
	Name         string
	Abbreviation string
}

type ListVolumeDiscountsParams struct {
	AccountID         string
	CustomerAccountID *string
	Cursor            *string
	Limit             int32
	Query             *string
}

type ListVolumeDiscountsResult struct {
	VolumeDiscounts []*VolumeDiscount
	PageInfo        pagination.PageInfo
}

type GetVolumeDiscountParams struct {
	AccountID         string
	VolumeDiscountID  string
	CustomerAccountID *string
}

type CreateVolumeDiscountTierParams struct {
	ID                 string
	Name               string
	DiscountPercentage string
	Threshold          string
	ParentTierID       *string
}

type CreateVolumeDiscountCustomerGroupParams struct {
	ID             string
	AccountGroupID string
}

type CreateVolumeDiscountParams struct {
	AccountID      string
	Name           string
	Tiers          []CreateVolumeDiscountTierParams
	CustomerGroups []CreateVolumeDiscountCustomerGroupParams
	ProductLineIDs []string
	CategoryIDs    []string
	AttributeIDs   []string
	UnitIDs        []string
}

type UpdateVolumeDiscountTierParams struct {
	ID                 *string
	GeneratedID        string
	Name               *string
	DiscountPercentage *string
	Threshold          *string
	ParentTierID       *string
}

type UpdateVolumeDiscountCustomerGroupParams struct {
	ID             string
	AccountGroupID string
}

type UpdateVolumeDiscountParams struct {
	AccountID         string
	VolumeDiscountID  string
	Name              *string
	Tiers             []UpdateVolumeDiscountTierParams
	CustomerGroups    []UpdateVolumeDiscountCustomerGroupParams
	ProductLineIDs    []string
	CategoryIDs       []string
	AttributeIDs      []string
	UnitIDs           []string
	HasTiers          bool
	HasCustomerGroups bool
	HasProductLines   bool
	HasCategories     bool
	HasAttributes     bool
	HasUnits          bool
}

type DeleteVolumeDiscountParams struct {
	AccountID        string
	VolumeDiscountID string
}
