package domain

import (
	"time"

	"github.com/augno/api/shared/pagination"
)

// AccountPrice represents a customer-specific price for a product line.
type AccountPrice struct {
	ID                   string
	OwnerAccountID       string
	RecipientAccountID   string `audit:"recipient_account_id"`
	RecipientAccountName string `audit:"recipient_account_name"`
	ProductLineID        string `audit:"product_line_id"`
	ProductLineName      string `audit:"product_line_name"`
	RateID               string
	RateValue            string                  `audit:"rate_value"`
	NumeratorUnitID      string                  `audit:"numerator_unit_id"`
	NumeratorUnitName    string                  `audit:"numerator_unit_name"`
	NumeratorUnitAbbr    string                  `audit:"numerator_unit_abbr"`
	NumeratorUnitType    string                  `audit:"numerator_unit_type"`
	DenominatorUnitID    string                  `audit:"denominator_unit_id"`
	DenominatorUnitName  string                  `audit:"denominator_unit_name"`
	DenominatorUnitAbbr  string                  `audit:"denominator_unit_abbr"`
	DenominatorUnitType  string                  `audit:"denominator_unit_type"`
	Categories           []AccountPriceCategory  `audit:"categories"`
	Attributes           []AccountPriceAttribute `audit:"attributes"`
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

// AccountPriceCategory represents a category association on an account price.
type AccountPriceCategory struct {
	ID   string
	Name string
}

// AccountPriceAttribute represents an attribute association on an account price.
type AccountPriceAttribute struct {
	ID    string
	Value string
}

type ListAccountPricesParams struct {
	AccountID          string
	Cursor             *string
	Limit              int32
	Query              *string
	RecipientAccountID *string
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
