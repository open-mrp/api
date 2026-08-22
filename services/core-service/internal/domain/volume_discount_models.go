package domain

import (
	"time"

	"github.com/open-mrp/api/shared/pagination"
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
	ID               string
	AccountGroupID   string
	Name             string
	CommissionPolicy string
	FreightPolicy    string
	Type             string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type VolumeDiscountProductLine struct {
	ID                 string
	Name               string
	IsCommissionExempt bool
	IsFreightExempt    bool
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type VolumeDiscountCategory struct {
	ID        string
	Name      string
	Type      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type VolumeDiscountAttribute struct {
	ID        string
	Name      string
	ColorCode string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type VolumeDiscountUnit struct {
	ID                string
	Name              string
	Abbreviation      string
	Type              string
	RatioNumerator    string
	RatioDenominator  string
	OffsetNumerator   string
	OffsetDenominator string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type ListVolumeDiscountsParams struct {
	AccountID         string
	CustomerAccountID *string
	Cursor            *string
	Limit             int32
	Query             *string
	Includes          []string
}

type ListVolumeDiscountsResult struct {
	VolumeDiscounts []*VolumeDiscount
	PageInfo        pagination.PageInfo
}

type GetVolumeDiscountParams struct {
	AccountID         string
	VolumeDiscountID  string
	CustomerAccountID *string
	Includes          []string
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
	Includes       []string
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
	Includes          []string
}

type DeleteVolumeDiscountParams struct {
	AccountID        string
	VolumeDiscountID string
}
