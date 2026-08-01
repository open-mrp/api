package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleDemandOverrideID = "deov_0192b7d38c4f5a9b02d3e16f88"
const SampleDemandOverrideTypeID = "deovtp_01seeddeltaunit"

// A way of adjusting planned demand.
//
// `absolute` replaces the forecast for the period, `delta_units` adds to it, and `delta_percent` scales it. When several overrides land on the same month they are applied in that order.
type DemandOverrideType struct {
	// Override type ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=demand_override_type"`
	// Stable code used when creating an override.
	Code constants.DemandOverrideAdjustment `json:"code" validate:"required"`
	// Display name of the type.
	Name string `json:"name" validate:"required"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleDemandOverrideType = &DemandOverrideType{
	ID:        SampleDemandOverrideTypeID,
	Object:    constants.ObjectTypeDemandOverrideType,
	Code:      constants.DemandOverrideAdjustmentDeltaUnits,
	Name:      "Delta Units",
	CreatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt: timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*DemandOverrideType) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleDemandOverrideType)
}

// An adjustment to the demand a production schedule plans against.
//
// Sales history cannot see a large customer that is about to order, a promotion, or a line that is being discontinued. An override is how management tells the planner about it. The period bounds the demand months the adjustment applies to; `effective_from` and `expires_at` bound when the override is consulted at all, which is a different question — an override for next quarter typically stops applying once the real orders arrive.
//
// A product-line override is distributed across the line's items in proportion to each item's baseline demand.
type DemandOverride struct {
	// Demand override ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=demand_override"`
	// What kind of resource the override targets. Mirrors `scope.type`, which is only present when the scope is expanded.
	ScopeType constants.DemandOverrideScope `json:"scope_type" validate:"required"`
	// The item or product line the override targets. Expandable.
	//
	// This is a single reference rather than one field per scope because a given override targets exactly one of them; `scope.type` names which.
	Scope *Entity `json:"scope" expandable:"true"`
	// First day of the demand period the override applies to.
	PeriodStartsAt time.Time `json:"period_starts_at" validate:"required"`
	// Last day of the demand period the override applies to.
	PeriodEndsAt time.Time `json:"period_ends_at" validate:"required"`
	// How the value adjusts the forecast.
	Adjustment constants.DemandOverrideAdjustment `json:"adjustment" validate:"required"`
	// The adjustment, interpreted according to `adjustment`.
	Value float64 `json:"value" validate:"required"`
	// The unit the value is expressed in. Expandable.
	Unit *Unit `json:"unit" expandable:"true"`
	// Why the adjustment was made.
	Reason *constants.DemandOverrideReason `json:"reason"`
	// Free-form notes about the adjustment.
	Note *string `json:"note"`
	// The actor that created the override. This may be a user, an API key, or an agent. Expandable.
	CreatedBy *Actor `json:"created_by" expandable:"true"`
	// When the override starts being applied to solves.
	EffectiveAt time.Time `json:"effective_at" validate:"required"`
	// When the override stops being applied to solves.
	ExpiresAt *time.Time `json:"expires_at"`
	// Whether the override is applied to solves at all.
	Status constants.ActivationStatus `json:"status" validate:"required"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var (
	sampleDemandOverrideReason = constants.DemandOverrideReasonNewCustomer
	sampleDemandOverrideNote   = "Northwind onboarding; first PO expected in September."
)

var SampleDemandOverride = &DemandOverride{
	ID:             SampleDemandOverrideID,
	Object:         constants.ObjectTypeDemandOverride,
	ScopeType:      constants.DemandOverrideScopeItem,
	Scope:          NewEntity(SampleItemID, constants.ObjectTypeItem, nil, nil),
	PeriodStartsAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	PeriodEndsAt:   timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
	Adjustment:     constants.DemandOverrideAdjustmentDeltaUnits,
	Value:          5000,
	Reason:         &sampleDemandOverrideReason,
	Note:           &sampleDemandOverrideNote,
	CreatedBy:      SampleActor,
	EffectiveAt:    timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	Status:         constants.ActivationStatusActive,
	CreatedAt:      timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:      timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*DemandOverride) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleDemandOverride)
}
