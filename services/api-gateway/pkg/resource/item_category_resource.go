package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleItemCategoryID = "ic_01jm4r6700f8nwq3v5hx2d9ktp"
const SampleItemCategoryName = "Electronics"
const SampleUnitGroupID = "ug_01jm4r6700f8nwq3v5hx2d9ktp"
const SampleUnitGroupName = "Weight"

// ItemCategory represents a full item category resource.
type ItemCategory struct {
	// The unique identifier for the item category.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=item_category"`
	// The display name of the item category.
	Name string `json:"name" validate:"required"`
	// Notes about the item category.
	Notes *string `json:"notes"`
	// The type of item category.
	Type constants.ItemCategoryType `json:"type" validate:"required"`
	// The owner of this resource.
	Owner *Owner `json:"owner" expandable:"true"`
	// The properties associated with this item category.
	Properties *List[Property] `json:"properties" expandable:"true"`
	// The unit group associated with this item category.
	UnitGroup *UnitGroup `json:"unit_group" expandable:"true"`
	// The timestamp when the item category was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// The timestamp when the item category was last updated.
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
