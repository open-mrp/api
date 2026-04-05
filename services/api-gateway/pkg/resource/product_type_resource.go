package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleProductTypeID = "prty_01jm4r6700f8nwq3v5hx2d9ktp"
const SampleProductTypeName = "Sale"
const SampleProductTypeCode = "sale"

// ProductType represents a product type that categorizes products.
type ProductType struct {
	// The unique identifier for the product type.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=product_type"`
	// The display name of the product type.
	Name string `json:"name" validate:"required"`
	// The unique code for the product type.
	Code string `json:"code" validate:"required"`
	// When this product type was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// When this product type was last updated.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleProductType = &ProductType{
	ID:        SampleProductTypeID,
	Object:    constants.ObjectTypeProductType,
	Name:      SampleProductTypeName,
	Code:      SampleProductTypeCode,
	CreatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt: timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*ProductType) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleProductType)
}
