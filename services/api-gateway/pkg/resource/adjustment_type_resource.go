package apiresource

import (
	"time"

	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/timeutil"
)

const SampleAdjustmentTypeID = "adjt_e2t6ruyqik5q"
const SampleAdjustmentTypeName = "Discount"
const SampleAdjustmentTypeCode = string(constants.AdjustmentTypeDiscount)

// A category of financial adjustment, such as a discount, fee, or write-off.
//
// Adjustment types classify the `adjustment` transactions recorded against a customer.
type AdjustmentType struct {
	// Adjustment type ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=adjustment_type"`
	// Human-readable name of the adjustment type (e.g. "Discount").
	Name string `json:"name" validate:"required"`
	// Machine-readable code identifying what kind of adjustment this is.
	//
	// - `discount`: a price reduction.
	// - `shipping_discrepancy`: corrects a difference between quoted and actual freight.
	// - `short_payment`: reconciles an invoice paid for less than the amount due.
	// - `write_off`: cancels an uncollectible balance.
	// - `fee`: an additional charge.
	// - `refund`: returns money to the customer.
	Code constants.AdjustmentType `json:"code" validate:"required"`
	// Owner of this resource.
	//
	// Adjustment types are platform-provided and shared across all accounts, so the owner is always the OpenMRP system owner.
	Owner *Owner `json:"owner" expandable:"true"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleAdjustmentType = &AdjustmentType{
	ID:        SampleAdjustmentTypeID,
	Object:    constants.ObjectTypeAdjustmentType,
	Name:      SampleAdjustmentTypeName,
	Code:      constants.AdjustmentTypeDiscount,
	Owner:     SampleOwnerSystem,
	CreatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt: timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*AdjustmentType) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleAdjustmentType)
}
