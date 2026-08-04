package demandoverridesep

import (
	"context"
	"net/http"
	"time"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// Request to create a demand override.
type CreateDemandOverrideRequest struct {
	// What the override targets.
	//
	// - `item`: a single item.
	// - `product_line`: every item sold under one product line.
	// - `account`: every item in the plan, which is how a blanket assumption such as "plan for double demand" is expressed.
	ScopeType constants.DemandOverrideScope `json:"scope_type" validate:"required"`
	// ID of the item or product line the override targets.
	//
	// Omit it for an `account`-wide override, which targets every planned item rather than one thing. The ID is checked against the account's items and product lines, so an override cannot be created against something that does not exist.
	ScopeRefID string `json:"scope_ref_id,omitzero" validate:"omitempty"`
	// First day of the demand period the override applies to.
	//
	// Overrides are applied month by month, so every calendar month the period touches is adjusted and any time of day is ignored.
	PeriodStartsAt time.Time `json:"period_starts_at" validate:"required"`
	// Last day of the demand period the override applies to.
	//
	// Must fall on or after `period_starts_at`.
	PeriodEndsAt time.Time `json:"period_ends_at" validate:"required"`
	// How the value adjusts the forecast.
	//
	// - `absolute`: replaces the forecast for each month in the period.
	// - `delta_units`: adds the value to each month in the period.
	// - `delta_percent`: scales each month in the period by the value as a percentage.
	//
	// When several overrides land on the same month they are applied in that order, so a percentage always acts on the already-adjusted number.
	Adjustment constants.DemandOverrideAdjustment `json:"adjustment" validate:"required"`
	// The amount of the adjustment, interpreted according to `adjustment`.
	//
	// A `delta_percent` value is a number of percent, so `-25` plans a quarter less than the forecast; it cannot go below `-100`. An `absolute` value cannot be negative, while a `delta_units` value can, so that a cancelled program removes demand.
	Value float64 `json:"value" validate:"required"`
	// ID of the unit the value is expressed in.
	//
	// Recorded for context only: the value is applied to the planned demand without unit conversion, so a unit adjustment should be stated in the unit the item is planned in.
	UnitID field.Optional[string] `json:"unit_id,omitzero"`
	// Why the adjustment was made.
	//
	// The reason is carried into each schedule the override changes, so a plan can explain why a month departs from history.
	Reason field.Optional[constants.DemandOverrideReason] `json:"reason,omitzero"`
	// Free-form notes about the adjustment.
	//
	// This is the text the free-text search on the list endpoint matches against.
	Note field.Optional[string] `json:"note,omitzero" validate:"omitempty,max=2000"`
	// When the override starts being applied to newly generated schedules.
	//
	// When omitted, the override starts applying straight away.
	EffectiveAt field.Optional[time.Time] `json:"effective_at,omitzero"`
	// When the override stops being applied to newly generated schedules.
	//
	// When omitted, the override keeps applying until it is deactivated or deleted.
	ExpiresAt field.Optional[time.Time] `json:"expires_at,omitzero"`
	// Whether the override is taken into account when a schedule is generated.
	//
	// Send `false` to stage an adjustment that should not affect schedules yet; an override is otherwise created ready to apply.
	Active field.Optional[bool] `json:"active,omitzero"`
}

var sampleCreateDemandOverrideRequest = &CreateDemandOverrideRequest{
	ScopeType:      constants.DemandOverrideScopeItem,
	ScopeRefID:     apiresource.SampleItemID,
	PeriodStartsAt: apiresource.SampleDemandOverride.PeriodStartsAt,
	PeriodEndsAt:   apiresource.SampleDemandOverride.PeriodEndsAt,
	Adjustment:     constants.DemandOverrideAdjustmentDeltaUnits,
	Value:          5000,
}

func (*CreateDemandOverrideRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateDemandOverrideRequest)
}

// Creates a demand override, telling the planner about demand the sales history cannot see.
//
// The scope reference is validated against the account's items and product lines, so an override can never silently match nothing. An `account`-scoped override takes no scope reference and must be a delta rather than an absolute value, since one number fanned out across every item would flatten the whole plan.
//
// Schedules that have already been generated are unaffected; the override is picked up by the next one.
type CreateDemandOverrideEndpoint struct{}

func (e *CreateDemandOverrideEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateDemandOverrideRequest, *apiresource.DemandOverride] {
	return (&apiendpoint.APIEndpoint[*CreateDemandOverrideRequest, *apiresource.DemandOverride]{
		Title:             "Create Demand Override",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/operations/demand-overrides",
		SuccessStatusCode: http.StatusCreated,
		Public:            true,
		Preview:           true,
		ObjectType:        constants.ObjectTypeDemandOverride,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainDemandOverrides, Action: types.ActionCreate},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateDemandOverrideRequest) (*apiresource.DemandOverride, *apierror.APIError) {
			return svc.(DemandOverridesSvc).CreateDemandOverride
		},
		LocationFunc: func(resp *apiresource.DemandOverride) string {
			return "/v1/operations/demand-overrides/" + resp.ID
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeDemandOverride,
			Fields:     []string{"scope"},
		}),
	})
}
