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

// ProductType resource.
type ProductType struct {
	// Product type ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=product_type"`
	// Display name.
	Name string `json:"name" validate:"required"`
	// Unique code.
	Code constants.ProductTypeCode `json:"code" validate:"required"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
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
