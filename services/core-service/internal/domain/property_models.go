package domain

import (
	"time"

	"github.com/open-mrp/apikit/pagination"
)

// Property represents a user-defined property that groups attributes.
type Property struct {
	ID         string
	Name       string `audit:"name"`
	AccountID  string
	IsPublic   bool `audit:"is_public"`
	Attributes []*Attribute
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// Attribute represents a value option within a property.
type Attribute struct {
	ID         string
	Value      string `audit:"value"`
	PropertyID string
	AccountID  string
	ColorCode  string `audit:"color_code"`
	SortOrder  int32  `audit:"sort_order"`
	IsPublic   bool   `audit:"is_public"`
	IsActive   bool   `audit:"is_active"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// CreatePropertyParams holds the parameters for creating a property.
type CreatePropertyParams struct {
	AccountID string
	Name      string
}

// UpdatePropertyParams holds the parameters for updating a property.
type UpdatePropertyParams struct {
	PropertyID string
	AccountID  string
	Name       *string
}

// ListPropertiesParams holds the parameters for listing properties.
type ListPropertiesParams struct {
	AccountID string
	Query     *string
	Cursor    *string
	Limit     int32
	// ItemCategoryIDs keeps properties attached to at least one of these categories; empty means no filter.
	ItemCategoryIDs []string
}

// ListPropertiesResult holds the result of listing properties.
type ListPropertiesResult struct {
	Properties []*Property
	PageInfo   pagination.PageInfo
}

// GetPropertyParams holds the parameters for getting a single property.
type GetPropertyParams struct {
	PropertyID string
	AccountID  string
}

// DeletePropertyParams holds the parameters for deleting a property.
type DeletePropertyParams struct {
	PropertyID string
	AccountID  string
}

// holds the fields for a single property in a bulk upsert
type UpsertPropertyParams struct {
	Name       string
	Attributes []UpsertPropertyAttributeParams
}

// holds one selectable value in a bulk property upsert; an absent color is assigned
type UpsertPropertyAttributeParams struct {
	Value     string
	ColorCode *string
}

// holds the input for a bulk upsert of properties
type BulkUpsertPropertiesParams struct {
	Properties []UpsertPropertyParams
}

// carries a bulk upsert row once normalized, so the job records a fully determined write
type ResolvedUpsertPropertyRow struct {
	Name       string
	Attributes []ResolvedUpsertPropertyAttribute
}

// carries one selectable value, swatch settled
type ResolvedUpsertPropertyAttribute struct {
	Value     string
	ColorCode string
}

// filters which properties land in an exported file
type ExportPropertiesParams struct {
	AccountID string
	Query     *string
}

// CreateAttributeParams holds the parameters for creating an attribute.
type CreateAttributeParams struct {
	Value      string
	PropertyID string
	AccountID  string
	ColorCode  string
	SortOrder  int32
	IsActive   bool
}

// UpdateAttributeParams holds the parameters for updating an attribute.
type UpdateAttributeParams struct {
	AttributeID string
	PropertyID  string
	AccountID   string
	Value       *string
	ColorCode   *string
	SortOrder   *int32
	IsActive    *bool
}

// ListAttributesParams holds the parameters for listing attributes.
type ListAttributesParams struct {
	AccountID  string
	PropertyID string
	Query      *string
	Cursor     *string
	Limit      int32
	// IsActive keeps attributes with that status; nil returns every status.
	IsActive *bool
}

// ListAttributesResult holds the result of listing attributes.
type ListAttributesResult struct {
	Attributes []*Attribute
	PageInfo   pagination.PageInfo
}

// GetAttributeParams holds the parameters for getting a single attribute.
type GetAttributeParams struct {
	AttributeID string
	PropertyID  string
	AccountID   string
}

// DeleteAttributeParams holds the parameters for deleting an attribute.
type DeleteAttributeParams struct {
	AttributeID string
	PropertyID  string
	AccountID   string
}

// PropertyReferences blocks deleting a property an item category still lists. Its attributes' own uses are
// checked separately, since they reference the attributes rather than the property.
var PropertyReferences = RegisterReferences(References{
	Noun: "property",
	Probes: []ReferenceProbe{
		{Label: "item categories", Table: "_item_categories_properties", Column: "B"},
	},
})

// AttributeReferences blocks deleting an attribute that is assigned or that scopes a price-list rule (legacy account
// prices and volume discounts mirror into rules).
var AttributeReferences = RegisterReferences(References{
	Noun:       "attribute",
	Archivable: true,
	Probes: []ReferenceProbe{
		{Label: "items", Table: "_item_attributes", Column: "A"},
		{Label: "price-list rules", Table: "price_list_rule_attribute", Column: "attribute_id"},
		{Label: "price-list rules", Table: "account_price_attribute", Column: "attribute_id"},
		{Label: "price-list rules", Table: "_quantity_discounts_attributes", Column: "A"},
	},
})
