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

// Request to update a demand override.
type UpdateDemandOverrideRequest struct {
	// ID of the demand override.
	DemandOverrideID string `path:"id" validate:"required"`
	// First day of the demand period the override applies to.
	PeriodStartsAt field.Optional[time.Time] `json:"period_starts_at,omitzero"`
	// Last day of the demand period the override applies to.
	PeriodEndsAt field.Optional[time.Time] `json:"period_ends_at,omitzero"`
	// How the value adjusts the forecast.
	Adjustment field.Optional[constants.DemandOverrideAdjustment] `json:"adjustment,omitzero"`
	// The adjustment, interpreted according to `adjustment`.
	Value field.Optional[float64] `json:"value,omitzero"`
	// ID of the unit the value is expressed in.
	UnitID field.Clearable[string] `json:"unit_id,omitzero"`
	// Why the adjustment was made.
	Reason field.Clearable[constants.DemandOverrideReason] `json:"reason,omitzero"`
	// Free-form notes about the adjustment.
	Note field.Clearable[string] `json:"note,omitzero" validate:"omitempty,max=2000"`
	// When the override stops being applied to solves. Clear it to make the override permanent.
	ExpiresAt field.Clearable[time.Time] `json:"expires_at,omitzero"`
	// Whether the override is applied to solves at all.
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
// The type and value are validated as a pair against the resulting override, so switching an existing units adjustment to `delta_percent` is checked as a percent even when only the type is sent.
type UpdateDemandOverrideEndpoint struct{}

func (e *UpdateDemandOverrideEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateDemandOverrideRequest, *apiresource.DemandOverride] {
	return (&apiendpoint.APIEndpoint[*UpdateDemandOverrideRequest, *apiresource.DemandOverride]{
		Title:             "Update Demand Override",
		Method:            http.MethodPatch,
		ContentType:       "application/json",
		Route:             "/v1/operations/demand-overrides/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
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
