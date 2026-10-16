package domain

import (
	"time"

	"github.com/open-mrp/apikit/pagination"

	"github.com/open-mrp/api/shared/constants"
)

// MaxPriceListRuleTiers bounds a tiered rule's breaks.
const MaxPriceListRuleTiers = 20

// MaxPriceListRuleScopeIDs bounds each scope dimension of a rule, so a rule reads back whole.
const MaxPriceListRuleScopeIDs = 100

// MaxPriceListAssignments bounds how many lists one customer or account group is assigned directly.
const MaxPriceListAssignments = 100

// PriceListEmbeddedRules is how many rules a price list carries when its rules are expanded: one more than an embedded
// list shows, so a longer list is noticed.
const PriceListEmbeddedRules = 11

// PriceList is a set of pricing rules assigned to customers or account groups.
type PriceList struct {
	ID          string
	AccountID   string
	Name        string                  `audit:"name"`
	Description *string                 `audit:"description"`
	Status      constants.ArchiveStatus `audit:"status"`
	Currency    constants.Currency
	// SourceType and SourceID name the legacy pricing record the list mirrors; nil for lists made as price lists.
	SourceType *constants.PriceListSourceType
	SourceID   *string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// PriceListRuleScope is the items a rule prices. An empty dimension matches any item; an item must have every
// attribute.
type PriceListRuleScope struct {
	ItemIDs        []string `audit:"item_ids"`
	ProductLineIDs []string `audit:"product_line_ids"`
	CategoryIDs    []string `audit:"category_ids"`
	AttributeIDs   []string `audit:"attribute_ids"`
}

// PriceListTier is one break of a tiered rule: from MinQuantity (in the rule's unit) up, PercentOfBase or UnitPrice.
type PriceListTier struct {
	MinQuantity   string     `audit:"min_quantity"`
	PercentOfBase *string    `audit:"percent_of_base"`
	UnitPrice     *UnitPrice `audit:"unit_price"`
}

// PriceListPrice is the price a rule sets.
type PriceListPrice struct {
	Type          constants.PriceListPriceType `audit:"type"`
	UnitPrice     *UnitPrice                   `audit:"unit_price"`
	PercentOfBase *string                      `audit:"percent_of_base"`
	// UnitID is a tiered rule's counting unit.
	UnitID string          `audit:"unit_id"`
	Tiers  []PriceListTier `audit:"tiers"`
}

// PriceListRule is one price for a scope of items.
type PriceListRule struct {
	ID          string
	PriceListID string
	AccountID   string
	Scope       PriceListRuleScope `audit:"scope"`
	Price       PriceListPrice     `audit:"price"`
	// SourceID names the legacy account price or volume discount a mirrored rule comes from.
	SourceID  *string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// PriceListRuleIDs is the first rules of a price list in list order, with the cursor after the last one shown when
// more exist.
type PriceListRuleIDs struct {
	IDs        []string
	NextCursor string
}

// ListPriceListsParams filters an account's price lists. IDs, when non-nil, restricts to those lists (an empty
// non-nil slice matches nothing).
type ListPriceListsParams struct {
	AccountID string
	Statuses  []constants.ArchiveStatus
	IDs       []string
	Query     *string
	Cursor    *string
	Limit     int32
	// CustomerIDs keeps lists that apply to any of these customers, directly or inherited.
	CustomerIDs []string
}

type ListPriceListsResult struct {
	PriceLists []*PriceList
	PageInfo   pagination.PageInfo
}

// ListPriceListRulesParams filters one list's rules. ItemIDs and ProductLineIDs keep rules whose scope names one of
// them.
type ListPriceListRulesParams struct {
	AccountID      string
	PriceListID    string
	ItemIDs        []string
	ProductLineIDs []string
	Cursor         *string
	Limit          int32
}

type ListPriceListRulesResult struct {
	Rules    []*PriceListRule
	PageInfo pagination.PageInfo
}

type CreatePriceListParams struct {
	Name        string
	Description *string
	Status      *constants.ArchiveStatus
	Currency    *string
}

type UpdatePriceListParams struct {
	PriceListID      string
	Name             *string
	Description      *string
	ClearDescription bool
	Status           *constants.ArchiveStatus
}

// PriceListRuleInput is a rule as requested: its scope and price.
type PriceListRuleInput struct {
	Scope PriceListRuleScope
	Price PriceListPrice
}

type CreatePriceListRuleParams struct {
	PriceListID string
	Rule        PriceListRuleInput
}

// UpdatePriceListRuleParams replaces the scope dimensions and price that are set.
type UpdatePriceListRuleParams struct {
	PriceListID    string
	RuleID         string
	ItemIDs        *[]string
	ProductLineIDs *[]string
	CategoryIDs    *[]string
	AttributeIDs   *[]string
	Price          *PriceListPrice
}

// PriceListScopeKind is one dimension of a rule's scope.
type PriceListScopeKind string

const (
	PriceListScopeItem        PriceListScopeKind = "item"
	PriceListScopeProductLine PriceListScopeKind = "product line"
	PriceListScopeCategory    PriceListScopeKind = "item category"
	PriceListScopeAttribute   PriceListScopeKind = "attribute"
	// PriceListScopeUnit is not a scope dimension: it checks the units a price names.
	PriceListScopeUnit PriceListScopeKind = "unit"
)

// PriceListSource ranks where a list applies to a buyer from, most binding first: the buyer's own (mirrored account
// prices), lists assigned to the buyer, its parent customer's, its account group's, then lists for every customer.
type PriceListSource int

const (
	PriceListSourceOwn PriceListSource = iota
	PriceListSourceDirect
	PriceListSourceParent
	PriceListSourceGroup
	PriceListSourceEveryone
)

// PricingRule is a rule that applies to the buyer, with where its list applies from.
type PricingRule struct {
	*PriceListRule
	Source PriceListSource
}

// PriceListAssignments maps a customer's account relation or an account group to its directly assigned list IDs.
type PriceListAssignments map[string][]string

// --- Legacy pricing records mirrored into price lists ------------------------

// LegacyAccountPrice is an account price as the legacy tables hold it.
type LegacyAccountPrice struct {
	ID                  string
	RecipientRelationID string
	RecipientAccountID  string
	RecipientName       string
	ProductLineID       string
	Value               string
	DenominatorUnitID   string
	CategoryIDs         []string
	AttributeIDs        []string
	CreatedAt           time.Time
}

// LegacyVolumeDiscount is a volume discount as the legacy tables hold it.
type LegacyVolumeDiscount struct {
	ID             string
	Name           string
	GroupIDs       []string
	ProductLineIDs []string
	CategoryIDs    []string
	AttributeIDs   []string
	UnitIDs        []string
	// Tiers in ascending threshold.
	Tiers     []LegacyVolumeTier
	CreatedAt time.Time
}

// LegacyVolumeTier is one threshold of a volume discount; Percentage is a fraction (0.05 = 5%).
type LegacyVolumeTier struct {
	Threshold  string
	Percentage string
}

// LegacyPricingGroup is an account group that carries volume discounts or price-group members.
type LegacyPricingGroup struct {
	ID          string
	Name        string
	Description *string
	// MemberRelationIDs are the account relations that list the group as a price group.
	MemberRelationIDs []string
}

// UnitIDs is every unit the price names: its per-unit, its counting unit and its tiers' per-units.
func (p PriceListPrice) UnitIDs() []string {
	var out []string
	if p.UnitPrice != nil && p.UnitPrice.PerUnitID != "" {
		out = append(out, p.UnitPrice.PerUnitID)
	}
	if p.UnitID != "" {
		out = append(out, p.UnitID)
	}
	for _, t := range p.Tiers {
		if t.UnitPrice != nil && t.UnitPrice.PerUnitID != "" {
			out = append(out, t.UnitPrice.PerUnitID)
		}
	}
	return out
}

// PriceListReferences blocks deleting a list still assigned to a customer or an account group.
var PriceListReferences = RegisterReferences(References{
	Noun:       "price list",
	Archivable: true,
	Probes: []ReferenceProbe{
		{Label: "customers", Table: "price_list_customer", Column: "price_list_id"},
		{Label: "account groups", Table: "price_list_account_group", Column: "price_list_id"},
	},
})
