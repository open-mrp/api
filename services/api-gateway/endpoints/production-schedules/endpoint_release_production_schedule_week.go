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
	//
	// Applied to every batch this release creates, across all machines in the week.
	ScanningStationID field.Optional[string] `json:"scanning_station_id,omitzero"`
	// Issue the whole week as new batches, leaving an earlier week's unworked lots where they are.
	//
	// Off unless you ask for it: reprinting a ticket the floor is already holding is exactly what carrying work forward exists to prevent, so it takes a deliberate choice to do it.
	SkipCarryForward field.Optional[bool] `json:"skip_carry_forward,omitzero"`
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
//
// Lots an earlier week already issued are carried forward rather than reissued. When a week fell short, the next plan asks for the shortfall — and the batches covering it are usually already printed and sitting on the floor. Those tickets are moved into this run and counted against the campaign, so only the genuinely new work is created. `carried_forward_batch_count` says how many arrived that way, and each one names the run it came off. Send `skip_carry_forward` to issue the whole week new instead.
//
// Cancelled campaigns and campaigns planned at zero are left behind rather than released. A week that would produce an implausible number of batches is rejected outright, since that is far more likely to be a misconfigured lot size than a real week's work.
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
