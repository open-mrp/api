package domain

import (
	"time"

	"github.com/augno/api/shared/pagination"
)

// AccountPrice represents a customer-specific price for a product line.
type AccountPrice struct {
	ID                               string
	OwnerAccountID                   string
	RecipientAccountID               string `audit:"recipient_account_id"`
	RecipientAccountName             string `audit:"recipient_account_name"`
	RecipientAccountNumber           string
	RecipientAccountStatus           string
	RecipientAccountIsEdiEnabled     bool
	RecipientAccountCommissionPolicy string
	RecipientAccountRelationshipType string
	RecipientAccountCreatedAt        time.Time
	RecipientAccountUpdatedAt        time.Time
	ProductLineID                    string `audit:"product_line_id"`
	ProductLineName                  string `audit:"product_line_name"`
	ProductLineIsCommissionExempt    bool
	ProductLineIsFreightExempt       bool
	ProductLineCreatedAt             time.Time
	ProductLineUpdatedAt             time.Time
	RateID                           string
	RateValue                        string `audit:"rate_value"`
	RateCreatedAt                    time.Time
	RateUpdatedAt                    time.Time
	NumeratorUnitID                  string `audit:"numerator_unit_id"`
	NumeratorUnitName                string `audit:"numerator_unit_name"`
	NumeratorUnitAbbr                string `audit:"numerator_unit_abbr"`
	NumeratorUnitType                string `audit:"numerator_unit_type"`
	NumeratorUnitRatioNumerator      string
	NumeratorUnitRatioDenominator    string
	NumeratorUnitOffsetNumerator     string
	NumeratorUnitOffsetDenominator   string
	NumeratorUnitCreatedAt           time.Time
	NumeratorUnitUpdatedAt           time.Time
	DenominatorUnitID                string `audit:"denominator_unit_id"`
	DenominatorUnitName              string `audit:"denominator_unit_name"`
	DenominatorUnitAbbr              string `audit:"denominator_unit_abbr"`
	DenominatorUnitType              string `audit:"denominator_unit_type"`
	DenominatorUnitRatioNumerator    string
	DenominatorUnitRatioDenominator  string
	DenominatorUnitOffsetNumerator   string
	DenominatorUnitOffsetDenominator string
	DenominatorUnitCreatedAt         time.Time
	DenominatorUnitUpdatedAt         time.Time
	Categories                       []AccountPriceCategory  `audit:"categories"`
	Attributes                       []AccountPriceAttribute `audit:"attributes"`
	CreatedAt                        time.Time
	UpdatedAt                        time.Time
}

// AccountPriceCategory represents a category association on an account price.
type AccountPriceCategory struct {
	ID        string
	Name      string
	Type      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// AccountPriceAttribute represents an attribute association on an account price.
type AccountPriceAttribute struct {
	ID        string
	Value     string
	ColorCode string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type ListAccountPricesParams struct {
	AccountID string
	Cursor    *string
	Limit     int32
	Query     *string
	// RecipientAccountIDs, when non-empty, restricts results to prices offered to one of
	// these accounts. The service expands a requested customer into that customer plus its
	// parent, since a price on the parent applies to orders its children place.
	RecipientAccountIDs []string
}

type ListAccountPricesResult struct {
	AccountPrices []*AccountPrice
	PageInfo      pagination.PageInfo
}

type CreateAccountPriceParams struct {
	AccountID             string
	RecipientAccountID    string
	ProductLineID         string
	RateValue             string
	RateNumeratorUnitID   string
	RateDenominatorUnitID string
	CategoryIDs           []string
	AttributeIDs          []string
}

type UpdateAccountPriceParams struct {
	AccountID             string
	AccountPriceID        string
	RecipientAccountID    *string
	ProductLineID         *string
	RateValue             *string
	RateNumeratorUnitID   *string
	RateDenominatorUnitID *string
	CategoryIDs           *[]string
	AttributeIDs          *[]string
}

// ExportPriceListParams names the customer whose price list is being exported.
type ExportPriceListParams struct {
	CustomerAccountID string
}
