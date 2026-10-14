package apiresource

import (
	"time"

	"github.com/open-mrp/api/shared/constants"
	apiexample "github.com/open-mrp/apikit/example"
	"github.com/open-mrp/apikit/timeutil"
)

const SampleItemCategoryID = "ic_d06g9c6yc9ck"
const SampleItemCategoryName = "Electronics"
const SampleUnitGroupID = "ug_andst6m79n41"
const SampleUnitGroupName = "Weight"

// The physical grouping of items: their kind, the properties they vary by, and their production defaults.
type ItemCategory struct {
	// Item category ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=item_category"`
	// Display name, such as `Fasteners`.
	//
	// Unique (case-insensitive) among your account's categories.
	Name string `json:"name" validate:"required"`
	// What kind of items the category holds.
	//
	// `material`: materials. `product`: products and parts. An item can only be in a category of its own kind. Can't change after creation.
	Type constants.ItemCategoryType `json:"type" validate:"required"`
	// Whether the category is a system category every account shares or one your account created.
	//
	// System categories can't be updated or deleted.
	Owner *Owner `json:"owner" expandable:"true"`
	// Properties the category's items vary by, such as `Color` or `Size`.
	//
	// Also shown in the customer catalog. Returns the first 10. When there are more, `page_info.next_page_url` fetches the rest.
	Properties *List[Property] `json:"properties" expandable:"true"`
	// Unit group the category's items are measured in, until items carry their own stocking unit.
	UnitGroup *UnitGroup `json:"unit_group" expandable:"true" visibility:"internal"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleItemCategory = &ItemCategory{
	ID:         SampleItemCategoryID,
	Object:     constants.ObjectTypeItemCategory,
	Name:       SampleItemCategoryName,
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
