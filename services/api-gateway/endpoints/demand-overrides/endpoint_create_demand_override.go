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
	ScopeType constants.DemandOverrideScope `json:"scope_type" validate:"required"`
	// ID of the item or product line the override targets.
	ScopeRefID string `json:"scope_ref_id" validate:"required"`
	// First day of the demand period the override applies to.
	PeriodStartsAt time.Time `json:"period_starts_at" validate:"required"`
	// Last day of the demand period the override applies to.
	PeriodEndsAt time.Time `json:"period_ends_at" validate:"required"`
	// How the value adjusts the forecast.
	Adjustment constants.DemandOverrideAdjustment `json:"adjustment" validate:"required"`
	// The adjustment, interpreted according to `adjustment`.
	Value float64 `json:"value" validate:"required"`
	// ID of the unit the value is expressed in.
	UnitID field.Optional[string] `json:"unit_id,omitzero"`
	// Why the adjustment was made.
	Reason field.Optional[constants.DemandOverrideReason] `json:"reason,omitzero"`
	// Free-form notes about the adjustment.
	Note field.Optional[string] `json:"note,omitzero" validate:"omitempty,max=2000"`
	// When the override starts being applied to solves. Defaults to now.
	EffectiveAt field.Optional[time.Time] `json:"effective_at,omitzero"`
	// When the override stops being applied to solves. Omit for an override with no end.
	ExpiresAt field.Optional[time.Time] `json:"expires_at,omitzero"`
	// Whether the override is applied to solves at all. Defaults to true.
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

// Creates a demand override.
//
// The scope reference is validated against the account's items or product lines, so an override can never silently match nothing. An `absolute` value replaces the forecast for the period, `delta_units` adds to it, and `delta_percent` scales it; a percent override cannot reduce demand by more than 100%.
type CreateDemandOverrideEndpoint struct{}

func (e *CreateDemandOverrideEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateDemandOverrideRequest, *apiresource.DemandOverride] {
	return (&apiendpoint.APIEndpoint[*CreateDemandOverrideRequest, *apiresource.DemandOverride]{
		Title:             "Create Demand Override",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/operations/demand-overrides",
		SuccessStatusCode: http.StatusCreated,
		Public:            false,
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
