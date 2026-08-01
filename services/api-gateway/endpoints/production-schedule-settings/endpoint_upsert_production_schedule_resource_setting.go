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
	// What kind of resource this overrides.
	ScopeType constants.ScheduleResourceScope `json:"scope_type" validate:"required"`
	// ID of the machine, department or production step.
	ScopeRefID string `json:"scope_ref_id" validate:"required"`
	// Whether this resource takes part in planning. Machines are selected by department, so this excludes one rather than opting one in.
	ParticipationStatus constants.ParticipationStatus `json:"participation_status" validate:"required"`
	// Weeks of lead time at this resource.
	LeadTimeWeeks field.Optional[float64] `json:"lead_time_weeks,omitzero" validate:"omitempty,gte=0"`
	// Weeks after the constraint campaign this resource's work starts.
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

// Writes a per-resource planning override.
//
// One override exists per resource, so this replaces any existing entry for the same scope rather than adding a second. Machines are selected by the constraint department, so this is where one is taken *out* of planning — a machine down for a rebuild — and where a department or step declares how many weeks after the constraint its work starts.
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
