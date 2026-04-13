package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleSysPropertyID = "sypp_01jm4r6700f8nwq3v5hx2d9ktp"
const SampleSysPropertyTypeID = "sypptp_01jm4r6700f8nwq3v5hx2d9ktp"
const SampleSysPropertyTypeName = "Transaction Number"
const SampleSysPropertyTypeCode = "transaction_number"
const SampleSysPropertyValueInt int32 = 42

// System property counter.
type SysProperty struct {
	// System property ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=sys_property"`
	// System property type.
	Type *SysPropertyType `json:"type" validate:"required"`
	// Current counter value.
	Value int32 `json:"value" validate:"required"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleSysProperty = &SysProperty{
	ID:     SampleSysPropertyID,
	Object: constants.ObjectTypeSysProperty,
	Type: &SysPropertyType{
		ID:     SampleSysPropertyTypeID,
		Object: constants.ObjectTypeSysPropertyType,
		Name:   SampleSysPropertyTypeName,
		Code:   SampleSysPropertyTypeCode,
	},
	Value:     SampleSysPropertyValueInt,
	CreatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt: timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*SysProperty) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleSysProperty)
}

// System property type.
type SysPropertyType struct {
	// System property type ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=sys_property_type"`
	// Display name.
	Name string `json:"name" validate:"required"`
	// Type code.
	Code string `json:"code" validate:"required"`
}

// System property value response.
type SysPropertyValue struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=sys_property_value"`
	// Counter value as a string.
	Value string `json:"value" validate:"required"`
}

var SampleSysPropertyValue = &SysPropertyValue{
	Object: constants.ObjectTypeSysPropertyValue,
	Value:  "42",
}

func (*SysPropertyValue) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleSysPropertyValue)
}
