package demandoverridesep

import (
	"context"
	"net/http"
	"time"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/field"
)

// Request to update a demand override.
type UpdateDemandOverrideRequest struct {
	// ID of the demand override.
	DemandOverrideID string `path:"id" validate:"required"`
	// First day of the demand period the override applies to.
	//
	// Overrides are applied month by month, so every calendar month the period touches is adjusted and any time of day is ignored.
	PeriodStartsAt field.Optional[time.Time] `json:"period_starts_at,omitzero"`
	// Last day of the demand period the override applies to.
	//
	// Must fall on or after the override's start, whether that is sent here or already stored.
	PeriodEndsAt field.Optional[time.Time] `json:"period_ends_at,omitzero"`
	// How the value adjusts the forecast.
	//
	// - `absolute`: replaces the forecast for each month in the period.
	// - `delta_units`: adds the value to each month in the period.
	// - `delta_percent`: scales each month in the period by the value as a percentage.
	Adjustment field.Optional[constants.DemandOverrideAdjustment] `json:"adjustment,omitzero"`
	// The amount of the adjustment, interpreted according to `adjustment`.
	//
	// It is validated against the adjustment the override ends up with, so switching a stored unit delta to `delta_percent` without sending a new value requires the existing value to be a legal percentage.
	Value field.Optional[float64] `json:"value,omitzero"`
	// ID of the unit the value is expressed in.
	//
	// Recorded for context only: the value is applied to the planned demand without unit conversion.
	UnitID field.Clearable[string] `json:"unit_id,omitzero"`
	// Why the adjustment was made.
	//
	// The reason is carried into each schedule the override changes, so a plan can explain why a month departs from history.
	Reason field.Clearable[constants.DemandOverrideReason] `json:"reason,omitzero"`
	// Free-form notes about the adjustment.
	Note field.Clearable[string] `json:"note,omitzero" validate:"omitempty,max=2000"`
	// When the override stops being applied to newly generated schedules.
	//
	// Clear it to keep the override applying until it is deactivated or deleted.
	ExpiresAt field.Clearable[time.Time] `json:"expires_at,omitzero"`
	// Whether the override is taken into account when a schedule is generated.
	//
	// Deactivating parks the override without losing it; it is skipped whatever its effective window says, and can be reactivated later.
	Active field.Optional[bool] `json:"active,omitzero"`
}

var sampleUpdateDemandOverrideRequest = &UpdateDemandOverrideRequest{
	DemandOverrideID: apiresource.SampleDemandOverrideID,
	Value:            field.Some(7500.0),
}

func (*UpdateDemandOverrideRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateDemandOverrideRequest)
}

// Updates a demand override.
//
// Only the fields sent are changed. The adjustment and value are validated as a pair against the resulting override, so switching a stored unit adjustment to `delta_percent` is checked as a percentage even when only the adjustment is sent; the period is checked the same way.
//
// What an override targets cannot be changed — create a new override to adjust a different item, product line, or the account as a whole. Schedules that have already been generated are unaffected; the change is picked up by the next one.
type UpdateDemandOverrideEndpoint struct{}

func (e *UpdateDemandOverrideEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateDemandOverrideRequest, *apiresource.DemandOverride] {
	return (&apiendpoint.APIEndpoint[*UpdateDemandOverrideRequest, *apiresource.DemandOverride]{
		Title:             "Update Demand Override",
		Method:            http.MethodPatch,
		ContentType:       "application/json",
		Route:             "/v1/operations/demand-overrides/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Preview:           true,
		ObjectType:        constants.ObjectTypeDemandOverride,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainDemandOverrides, Action: types.ActionUpdate},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateDemandOverrideRequest) (*apiresource.DemandOverride, *apierror.APIError) {
			return svc.(DemandOverridesSvc).UpdateDemandOverride
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeDemandOverride,
			Fields:     []string{"scope"},
		}),
	})
}
