package apiresource

import (
	"time"

	"github.com/open-mrp/api/shared/constants"
	apiexample "github.com/open-mrp/apikit/example"
	"github.com/open-mrp/apikit/timeutil"
)

const SampleAccountGroupID = "acgp_6p4z57e9alaf"
const SampleAccountGroupName = "Wholesale Customers"

// A named grouping of customer accounts, used for pricing rules or to categorize accounts.
//
// A customer carries at most one group of type `type_group` as its customer type, plus any number of groups of type `pricing_group`. Membership of either kind can scope a volume discount to the customer and open up product lines for it to order from.
type AccountGroup struct {
	// Account group ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=account_group"`
	// How this account group is used.
	//
	// - `pricing_group`: used for pricing rules, such as a "Preferred" group that receives a special discount.
	// - `type_group`: used to categorize accounts, such as "Consumers" or "Distributors".
	//
	// A group's type is fixed when it is created and cannot be changed afterwards.
	Type constants.AccountGroupType `json:"type" validate:"required"`
	// Display name of the account group.
	//
	// Unique within the account.
	Name string `json:"name" validate:"required"`
	// Free-form description of the account group.
	Description *string `json:"description"`
	// How sales commission applies to accounts in this group.
	//
	// - `applied`: sales commission is calculated on orders from accounts in this group.
	// - `exempt`: orders from accounts in this group are exempt from commission.
	//
	// Null to customer and supplier portal users, like the rest of your commission settings.
	CommissionPolicy *constants.CommissionPolicy `json:"commission_policy" validate:"required" sensitive:"internal"`
	// How freight charges apply to orders from accounts in this group.
	//
	// - `free`: customers within this group will not have to pay for freight.
	// - `billed`: freight will be applied to any order within this account group, unless overridden elsewhere.
	FreightPolicy constants.FreightPolicy `json:"freight_policy" validate:"required"`
	// Calendar days between an order being issued and it being due to ship, inherited by every customer in this group that has neither set its own nor inherited one from a parent account.
	DefaultLeadTimeDays *int32 `json:"default_lead_time_days"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleAccountGroup = &AccountGroup{
	ID:               SampleAccountGroupID,
	Object:           constants.ObjectTypeAccountGroup,
	Type:             constants.AccountGroupTypeTypeGroup,
	Name:             SampleAccountGroupName,
	Description:      nil,
	CommissionPolicy: new(constants.CommissionPolicyApplied),
	FreightPolicy:    constants.FreightPolicyBilled,
	CreatedAt:        timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:        timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*AccountGroup) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleAccountGroup)
}
