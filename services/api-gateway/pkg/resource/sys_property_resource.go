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

// SysProperty represents a system property counter.
type SysProperty struct {
	// The unique identifier for the system property.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=sys_property"`
	// The system property type.
	Type *SysPropertyType `json:"type" validate:"required"`
	// The current counter value.
	Value int32 `json:"value" validate:"required"`
	// The timestamp when the system property was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// The timestamp when the system property was last updated.
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

// SysPropertyType represents a system property type.
type SysPropertyType struct {
	// The unique identifier for the system property type.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=sys_property_type"`
	// The display name of the type.
	Name string `json:"name" validate:"required"`
	// The code of the type.
	Code string `json:"code" validate:"required"`
}

// SysPropertyValue represents a system property value response.
type SysPropertyValue struct {
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=sys_property_value"`
	// The string value.
	Value string `json:"value" validate:"required"`
}

var SampleSysPropertyValue = &SysPropertyValue{
	Object: constants.ObjectTypeSysPropertyValue,
	Value:  "42",
}

func (*SysPropertyValue) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleSysPropertyValue)
}
