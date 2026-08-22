package apiresource

import (
	"time"

	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/timeutil"
)

const SampleDemandOverrideID = "deov_p8roudstrung"
const SampleDemandOverrideTypeID = "deovtp_z8ir1rabbsmt"

// A way of adjusting planned demand.
//
// `absolute` replaces the forecast for each month an override covers, `delta_units` adds to it, and `delta_percent` scales it. When several overrides land on the same month they are applied in that order.
type DemandOverrideType struct {
	// Override type ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=demand_override_type"`
	// The value to send as an override's `adjustment`.
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

// An adjustment to the demand a production schedule is planned against.
//
// Sales history cannot see a large customer that is about to order, a promotion, or a line that is being discontinued. An override is how management tells the planner about it. The period names the months the demand will occur in, and only months of the coming planning year are adjusted — a period entirely in the past changes nothing, because the plan covers the year ahead. `effective_at` and `expires_at` answer a different question: how long the override is consulted at all, so an adjustment can be retired on a date without deleting it.
type DemandOverride struct {
	// Demand override ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=demand_override"`
	// What the override targets.
	//
	// - `item`: a single item.
	// - `product_line`: every item sold under one product line.
	// - `account`: every item in the plan, which is how a blanket assumption such as "plan for double demand" is expressed.
	ScopeType constants.DemandOverrideScope `json:"scope_type" validate:"required"`
	// The item or product line the override targets.
	//
	// An account-wide override has no scope resource, because it targets every planned item rather than one thing.
	Scope *Entity `json:"scope" expandable:"true"`
	// First day of the demand period the override applies to.
	//
	// Overrides are applied month by month, so every calendar month the period touches is adjusted and any time of day is ignored.
	PeriodStartsAt time.Time `json:"period_starts_at" validate:"required"`
	// Last day of the demand period the override applies to.
	PeriodEndsAt time.Time `json:"period_ends_at" validate:"required"`
	// How the value adjusts the forecast.
	//
	// - `absolute`: replaces the forecast for each month in the period.
	// - `delta_units`: adds the value to each month in the period.
	// - `delta_percent`: scales each month in the period by the value as a percentage.
	//
	// When several overrides land on the same month they are applied in that order, so a percentage always acts on the already-adjusted number. An adjusted month is never taken below zero.
	Adjustment constants.DemandOverrideAdjustment `json:"adjustment" validate:"required"`
	// The amount of the adjustment, interpreted according to `adjustment`.
	//
	// A `delta_percent` value is a number of percent, so `-25` plans a quarter less than the forecast.
	Value float64 `json:"value" validate:"required"`
	// The unit the value is expressed in.
	//
	// Recorded for context only: the value is applied to the planned demand without unit conversion, so a unit adjustment should be stated in the unit the item is planned in.
	Unit *Unit `json:"unit" expandable:"true"`
	// Why the adjustment was made.
	//
	// The reason is carried into each schedule the override changes, so a plan can explain why a month departs from history.
	Reason *constants.DemandOverrideReason `json:"reason"`
	// Free-form notes about the adjustment.
	Note *string `json:"note"`
	// The actor that created the override.
	//
	// May be a user, an API key, or an agent.
	CreatedBy *Actor `json:"created_by" expandable:"true"`
	// When the override starts being applied to newly generated schedules.
	EffectiveAt time.Time `json:"effective_at" validate:"required"`
	// When the override stops being applied to newly generated schedules.
	//
	// An override with no expiry keeps applying until it is deactivated or deleted.
	ExpiresAt *time.Time `json:"expires_at"`
	// Whether the override is taken into account when a schedule is generated.
	//
	// An inactive override is skipped whatever its effective window says, which is how a prepared adjustment is parked without losing it.
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

// The demand period the sample override adjusts: the September–November quarter the new customer's orders are expected to land in.
const sampleDemandOverridePeriodStart = "2026-09-01T00:00:00Z"
const sampleDemandOverridePeriodEnd = "2026-11-30T00:00:00Z"

// The override stops being consulted once the period it adjusts has passed.
const sampleDemandOverrideExpiry = "2026-12-01T00:00:00Z"

var SampleDemandOverride = &DemandOverride{
	ID:             SampleDemandOverrideID,
	Object:         constants.ObjectTypeDemandOverride,
	ScopeType:      constants.DemandOverrideScopeItem,
	Scope:          NewEntity(SampleItemID, constants.ObjectTypeItem, nil, new(SampleItemSKU)),
	PeriodStartsAt: timeutil.TimestampToTime(sampleDemandOverridePeriodStart),
	PeriodEndsAt:   timeutil.TimestampToTime(sampleDemandOverridePeriodEnd),
	Adjustment:     constants.DemandOverrideAdjustmentDeltaUnits,
	Value:          5000,
	Unit:           newSampleUnit("Pair", "pr", constants.UnitTypeQuantity),
	Reason:         &sampleDemandOverrideReason,
	Note:           &sampleDemandOverrideNote,
	CreatedBy:      SampleActor,
	EffectiveAt:    timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	ExpiresAt:      timeutil.TimestampToTimePtr(sampleDemandOverrideExpiry),
	Status:         constants.ActivationStatusActive,
	CreatedAt:      timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:      timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*DemandOverride) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleDemandOverride)
}
