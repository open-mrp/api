package apiresource

import (
	"time"

	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/timeutil"
)

const SampleItemCategoryID = "ic_d06g9c6yc9ck"
const SampleItemCategoryName = "Electronics"
const SampleUnitGroupID = "ug_andst6m79n41"
const SampleUnitGroupName = "Weight"

// A grouping of related catalog items that defines the unit group and properties available to the items within it.
type ItemCategory struct {
	// Item category ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=item_category"`
	// Display name of the item category.
	Name string `json:"name" validate:"required"`
	// Free-form notes about the item category.
	Notes *string `json:"notes"`
	// What kind of items this category groups.
	//
	// - `material_category`: groups raw materials and components (items of type `material`).
	// - `product_category`: groups finished products and parts (items of type `product` or `part`).
	//
	// An item can only be assigned to a category whose type matches the item's `type`, and the category's type is fixed at creation.
	Type constants.ItemCategoryType `json:"type" validate:"required"`
	// Provenance of the item category.
	//
	// System-owned categories are platform-provided defaults shared across all accounts and cannot be updated or deleted; account-owned categories are custom to your account.
	Owner *Owner `json:"owner" expandable:"true"`
	// Properties associated with this item category, such as `Color` or `Size`.
	//
	// These describe the dimensions along which items in the category vary, and are also what the customer-facing catalog shows for the category. Attach and detach them with the Add Item Category Property and Remove Item Category Property endpoints.
	Properties *List[Property] `json:"properties" expandable:"true"`
	// Unit group associated with this item category.
	//
	// Items in this category are measured in units belonging to this group, and can only be ordered in those units unless the item's product line defines its own unit group, which takes precedence.
	UnitGroup *UnitGroup `json:"unit_group" expandable:"true"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var sampleItemCategoryNotes = "Components and raw materials used across the electronics assembly line."

var SampleItemCategory = &ItemCategory{
	ID:         SampleItemCategoryID,
	Object:     constants.ObjectTypeItemCategory,
	Name:       SampleItemCategoryName,
	Notes:      &sampleItemCategoryNotes,
	Type:       constants.ItemCategoryTypeMaterial,
	Owner:      SampleOwnerSystem,
	Properties: NewList([]Property{*SampleProperty}, PageInfo{}),
	UnitGroup:  SampleUnitGroup,
	CreatedAt:  timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:  timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*ItemCategory) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleItemCategory)
}
