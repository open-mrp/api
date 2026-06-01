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

// ItemCategory resource.
type ItemCategory struct {
	// Item category ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=item_category"`
	// Display name.
	Name string `json:"name" validate:"required"`
	// Notes.
	Notes *string `json:"notes"`
	// Item category type.
	Type constants.ItemCategoryType `json:"type" validate:"required"`
	// Owner of the item category.
	Owner *Owner `json:"owner" expandable:"true"`
	// Properties associated with this item category.
	Properties *List[Property] `json:"properties" expandable:"true"`
	// Unit group associated with this item category. This unit group dictates the available units that items in this category may embody in your production process.
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
