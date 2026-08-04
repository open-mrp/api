package productionschedulesettingsep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// Request to write a per-resource planning override.
type UpsertResourceSettingRequest struct {
	// What kind of resource this override applies to.
	//
	// Together with the resource ID it identifies the override, so writing the same pair again updates the existing entry in place and keeps its ID.
	ScopeType constants.ScheduleResourceScope `json:"scope_type" validate:"required"`
	// ID of the machine, department or production step being overridden, matching the scope type.
	ScopeRefID string `json:"scope_ref_id" validate:"required"`
	// Whether this resource takes part in planning.
	//
	// Machines are chosen by naming the constraint department, so this is how one is taken out — a machine down for a rebuild — rather than how one is opted in.
	ParticipationStatus constants.ParticipationStatus `json:"participation_status" validate:"required"`
	// Weeks of lead time at this resource.
	LeadTimeWeeks field.Optional[float64] `json:"lead_time_weeks,omitzero" validate:"omitempty,gte=0"`
	// How many weeks after the step feeding it this resource's work starts.
	//
	// Read when downstream department work is derived from the constraint plan, so it is the production-step override that shifts a plan: without an offset every step lands in the same week as the step feeding it, and the offsets along a chain of steps add up. A schedule is planned in whole weeks, so a fractional offset is truncated.
	LeadTimeOffsetWeeks float64 `json:"lead_time_offset_weeks" validate:"gte=0"`
}

var sampleUpsertResourceSettingRequest = &UpsertResourceSettingRequest{
	ScopeType:           constants.ScheduleResourceScopeMachine,
	ScopeRefID:          apiresource.SampleMachineID,
	ParticipationStatus: constants.ParticipationStatusExcluded,
}

func (*UpsertResourceSettingRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpsertResourceSettingRequest)
}

// Writes a planning override for one machine, department or production step.
//
// A resource has at most one override, so this replaces the existing entry for the same scope rather than adding a second, and the entry keeps the ID it already had. Machines are chosen by naming the constraint department, so this is where one is taken *out* of planning — a machine down for a rebuild — and where a production step declares how many weeks its work starts after the step that feeds it.
//
// Overrides are read when a plan is generated, so a change takes effect on the next generated version and leaves existing ones untouched.
type UpsertResourceSettingEndpoint struct{}

func (e *UpsertResourceSettingEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpsertResourceSettingRequest, *apiresource.ProductionScheduleResourceSetting] {
	return (&apiendpoint.APIEndpoint[*UpsertResourceSettingRequest, *apiresource.ProductionScheduleResourceSetting]{
		Title:             "Upsert Production Schedule Resource Setting",
		Method:            http.MethodPut,
		ContentType:       "application/json",
		Route:             "/v1/operations/production-schedule-settings/resources",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ObjectType:        constants.ObjectTypeProductionScheduleResourceSetting,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainProductionSchedules, Action: types.ActionUpdate},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpsertResourceSettingRequest) (*apiresource.ProductionScheduleResourceSetting, *apierror.APIError) {
			return svc.(ProductionScheduleSettingsSvc).UpsertResourceSetting
		},
	})
}
