package apiresource

import (
	"time"

	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/timeutil"
)

const SampleSysPropertyID = "sypp_1czynnv1b8kc"
const SampleSysPropertyTypeID = "sypptp_qxmnkeq1ig5c"
const SampleSysPropertyTypeName = "Transaction Number"
const SampleSysPropertyTypeCode = "transaction_number"
const SampleSysPropertyValueInt int32 = 42

// A counter maintained by the system for a numbered series, such as transaction or sales order numbers.
//
// Each account keeps at most one counter per counter type, created the first time that number series is used.
type SysProperty struct {
	// System property ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=sys_property"`
	// The kind of counter this property tracks.
	Type *SysPropertyType `json:"type" validate:"required"`
	// The counter's current position in its number series.
	//
	// The system advances the counter as it hands out numbers, so this normally matches the most recent number assigned in the series rather than the next one to be issued.
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
	// Machine-readable code identifying which number series this counter feeds.
	//
	// - `transaction_number`: numbering for financial transactions such as payments, credit memos, adjustments, and rebates.
	// - `settlement_number`: numbering for settlements that apply transactions to invoices.
	// - `sales_order_number`: numbering for sales orders.
	// - `purchase_order_number`: numbering for purchase orders.
	// - `customer_number`: identifiers assigned to new customers.
	// - `supplier_number`: identifiers assigned to new suppliers.
	// - `production_run_number`: numbering for production runs.
	// - `sscc_count`: serial component of the GS1 SSCC-18 codes assigned to shipping cases.
	Code constants.SysPropertyTypeCode `json:"code" validate:"required"`
}

// The value read from a system property counter.
type SysPropertyValue struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=sys_property_value"`
	// The number the counter holds after this read.
	Value string `json:"value" validate:"required"`
}

var SampleSysPropertyValue = &SysPropertyValue{
	Object: constants.ObjectTypeSysPropertyValue,
	Value:  "42",
}

func (*SysPropertyValue) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleSysPropertyValue)
}
