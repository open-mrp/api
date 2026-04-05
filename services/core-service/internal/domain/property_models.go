package domain

import (
	"time"

	"github.com/augno/api/shared/pagination"
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

// CreateAttributeParams holds the parameters for creating an attribute.
type CreateAttributeParams struct {
	Value      string
	PropertyID string
	AccountID  string
	ColorCode  string
	SortOrder  int32
}

// UpdateAttributeParams holds the parameters for updating an attribute.
type UpdateAttributeParams struct {
	AttributeID string
	PropertyID  string
	AccountID   string
	Value       *string
	ColorCode   *string
	SortOrder   *int32
}

// ListAttributesParams holds the parameters for listing attributes.
type ListAttributesParams struct {
	AccountID  string
	PropertyID string
	Query      *string
	Cursor     *string
	Limit      int32
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
