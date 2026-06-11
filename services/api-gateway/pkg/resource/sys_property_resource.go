package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleSysPropertyID = "sypp_01d8fd3a8b1a8e4c41be55ab5a"
const SampleSysPropertyTypeID = "sypptp_0197d530307d69870d9b6fc97f"
const SampleSysPropertyTypeName = "Transaction Number"
const SampleSysPropertyTypeCode = "transaction_number"
const SampleSysPropertyValueInt int32 = 42

// Monotonic counter maintained by the system, such as the next transaction or document number to assign.
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

// The kind of counter a system property tracks.
type SysPropertyType struct {
	// System property type ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=sys_property_type"`
	// Human-readable name of the counter, such as `Transaction Number`.
	Name string `json:"name" validate:"required"`
	// Machine-readable code identifying which counter this is, such as `transaction_number` or `purchase_order_number`.
	Code string `json:"code" validate:"required"`
}

// The current value of a system property counter.
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
