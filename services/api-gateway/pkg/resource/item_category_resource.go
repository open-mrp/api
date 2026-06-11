package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleItemCategoryID = "ic_01ae7bd7bfd21ca0ab81e1357e"
const SampleItemCategoryName = "Electronics"
const SampleUnitGroupID = "ug_01aad07abb8e41fd392d2d7013"
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
	// An item can only be assigned to a category whose type matches the item's `type`.
	//
	// - `material_category`: groups raw materials and components (items of type `material`).
	// - `product_category`: groups finished products and parts (items of type `product` or `part`).
	Type constants.ItemCategoryType `json:"type" validate:"required"`
	// Owner of the item category.
	//
	// System-owned categories are platform defaults (the `owner.type` is `system` and `owner.account` is `null`); account-owned categories were created by your organization.
	Owner *Owner `json:"owner" expandable:"true"`
	// Properties associated with this item category.
	Properties *List[Property] `json:"properties" expandable:"true"`
	// Unit group associated with this item category.
	//
	// This unit group determines the units of measure available to items in this category throughout your production process.
	UnitGroup *UnitGroup `json:"unit_group" expandable:"true"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleItemCategory = &ItemCategory{
	ID:        SampleItemCategoryID,
	Object:    constants.ObjectTypeItemCategory,
	Name:      SampleItemCategoryName,
	Type:      constants.ItemCategoryTypeMaterial,
	Owner:     SampleOwnerSystem,
	CreatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt: timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*ItemCategory) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleItemCategory)
}
