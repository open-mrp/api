package productionscheduleep

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

// Request to release one week of a production schedule to the floor.
type ReleaseProductionScheduleWeekRequest struct {
	// ID of the production schedule.
	ProductionScheduleID string `path:"id" validate:"required"`
	// Zero-based week offset from the start of the horizon.
	WeekIndex int32 `json:"week_index" validate:"min=0"`
	// ID of the account user accountable for executing the run.
	//
	// Accepts either an account user ID or a user ID; it is resolved and stored as the account user.
	ResponsibleUserID string `json:"responsible_user_id" validate:"required"`
	// ID of the scanning station the batches will be scanned at.
	ScanningStationID field.Optional[string] `json:"scanning_station_id,omitzero"`
}

var sampleReleaseProductionScheduleWeekRequest = &ReleaseProductionScheduleWeekRequest{
	WeekIndex:         0,
	ResponsibleUserID: apiresource.SampleUserID,
}

func (*ReleaseProductionScheduleWeekRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleReleaseProductionScheduleWeekRequest)
}

// Turns one planned week into a production run.
//
// Each campaign in the week becomes one batch per planned lot, using the lot size the campaign was planned at. A 360-unit campaign at a 60-unit lot arrives on the floor as six batches, not one instruction to make 360; a quantity that is not a whole number of lots trails a single short lot at the end of the run.
//
// The release is atomic. A run holding half a week's batches is worse than no run, because the missing half looks like work nobody was asked to do and attainment would count it as unplanned production.
//
// Releasing the same week twice fails rather than creating a second run. Each released line records the run now carrying it, and a line that is already released is never re-pointed.
type ReleaseProductionScheduleWeekEndpoint struct{}

func (e *ReleaseProductionScheduleWeekEndpoint) Materialize() *apiendpoint.APIEndpoint[*ReleaseProductionScheduleWeekRequest, *apiresource.ReleaseScheduleWeekResult] {
	return (&apiendpoint.APIEndpoint[*ReleaseProductionScheduleWeekRequest, *apiresource.ReleaseScheduleWeekResult]{
		Title:             "Release Production Schedule Week",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/operations/production-schedules/{id}/actions/release-week",
		SuccessStatusCode: http.StatusCreated,
		Public:            true,
		Preview:           true,
		ObjectType:        constants.ObjectTypeProductionScheduleWeekRelease,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainProductionSchedules, Action: types.ActionUpdate},
			{Domain: types.PermissionDomainProductionRuns, Action: types.ActionCreate},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ReleaseProductionScheduleWeekRequest) (*apiresource.ReleaseScheduleWeekResult, *apierror.APIError) {
			return svc.(ProductionScheduleSvc).ReleaseProductionScheduleWeek
		},
	})
}
