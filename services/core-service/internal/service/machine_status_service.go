package service

import (
	"context"
	"fmt"
	"time"

	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/tracing"
)

var machineStatusSvcTracer = tracing.GetTracer("core-service.machine_status_service")

type machineStatusSvcImpl struct {
	repos domain.RepoFactory
}

type MachineStatusSvcConfig struct {
	// Repos (required) is the repository factory.
	Repos domain.RepoFactory
}

func (c *MachineStatusSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("machine status service: repos is required")
	}
	return nil
}

func NewMachineStatusSvc(config *MachineStatusSvcConfig) domain.MachineStatusSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}
	return &machineStatusSvcImpl{repos: config.Repos}
}

// ListMachineStatus is what the floor and the office both read: what each machine is on, how much is left, what is queued, and whether it is down.
//
// The floor is assembled from three reads: the machines, the plan from this week forward, and whatever is down right now. The assembly happens here rather than in SQL because "current" and "next" are decided by scan progress, and expressing that as a window function would make the query far harder to reason about than the loop it replaces.
func (s *machineStatusSvcImpl) ListMachineStatus(ctx context.Context, params domain.ListMachineStatusParams) (*domain.MachineStatusResult, *apierror.APIError) {
	ctx, span := machineStatusSvcTracer.Start(ctx, "service.machine_status.list")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	// Gated on machines rather than schedules: this is a view of the plant, and a supervisor who can see the machines should see what they are running without also being granted the plan.
	if apiErr := identity.CheckHasPermission(types.PermissionDomainMachines, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID
	if params.AsOf.IsZero() {
		params.AsOf = time.Now().UTC()
	}

	// weekStart normalizes to the Monday of the week, matching how the solver lays out a horizon and how scans are bucketed.
	weekStart := weekStart(params.AsOf)

	result := &domain.MachineStatusResult{
		WeekStartDate: weekStart,
		Machines:      []domain.MachineStatus{},
	}

	repo := s.repos.NewMachineStatusRepo()

	machines, apiErr := repo.ListMachinesForStatus(ctx, params.AccountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	downRows, apiErr := repo.ListOpenDowntimeForStatus(ctx, params.AccountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	downByMachine := map[string]*domain.MachineDowntimeSummary{}
	for _, row := range downRows {
		summary := &domain.MachineDowntimeSummary{
			EventID:   row.ID,
			Reason:    row.ReasonCode,
			StartedAt: row.StartedAt,
		}
		if row.ReasonName != nil {
			summary.ReasonName = *row.ReasonName
		}
		if row.OEEBucket != nil {
			summary.OEEBucket = *row.OEEBucket
		}
		summary.Note = row.Note
		downByMachine[row.MachineID] = summary
	}

	// The published version is what the floor works to. Nothing published is a normal state — planning may not have caught up — so every machine reads idle rather than the endpoint failing.
	schedule, apiErr := s.repos.NewProductionScheduleRepo().GetCurrent(ctx, params.AccountID, params.AsOf)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	linesByMachine := map[string][]domain.MachineCampaign{}
	if schedule != nil {
		result.ProductionScheduleID = schedule.ID

		rows, apiErr := repo.ListScheduleLinesForStatus(ctx, domain.ListScheduleLinesForStatusParams{
			AccountID:            params.AccountID,
			ProductionScheduleID: schedule.ID,
			FromWeek:             weekStart,
		})
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}

		for _, row := range rows {
			remaining := row.PlannedQuantity - row.ScannedQuantity
			if remaining < 0 {
				remaining = 0
			}

			campaign := domain.MachineCampaign{
				ProductionScheduleLineID: row.ID,
				ItemID:                   row.ItemID,
				WeekStartDate:            row.WeekStartDate,
				WeekIndex:                row.WeekIndex,
				PlannedQuantity:          row.PlannedQuantity,
				ScannedQuantity:          row.ScannedQuantity,
				RemainingQuantity:        remaining,
				ReleasedBatchCount:       row.ReleasedBatchCount,
				ScannedBatchCount:        row.ScannedBatchCount,
				PlannedRunHours:          row.PlannedRunHours,
				StatusCode:               row.StatusCode,
			}
			if row.SKU != nil {
				campaign.SKU = *row.SKU
			} else {
				// No policy row for the item; the id keeps the campaign identifiable.
				campaign.SKU = row.ItemID
			}
			if row.PlannedUnitAbbreviation != nil {
				campaign.Unit = *row.PlannedUnitAbbreviation
			}
			campaign.ProductionRunID = row.ProductionRunID

			linesByMachine[row.MachineID] = append(linesByMachine[row.MachineID], campaign)
		}
	}

	departmentFilter := toSet(params.DepartmentIDs)

	for _, machine := range machines {
		// department_id is NOT NULL on machine, so an empty string is the "unassigned" case.
		var departmentID *string
		if machine.DepartmentID != "" {
			id := machine.DepartmentID
			departmentID = &id
		}
		if len(departmentFilter) > 0 && (departmentID == nil || !passesFilter(departmentFilter, *departmentID)) {
			continue
		}

		status := domain.MachineStatus{
			MachineID:    machine.ID,
			MachineName:  machine.Name,
			DepartmentID: departmentID,
			Status:       constants.MachineWorkStatusIdle,
		}
		status.DepartmentName = machine.DepartmentName

		campaigns := linesByMachine[machine.ID]
		current, next := currentAndNext(campaigns, weekStart)
		status.Current = current
		status.Next = next

		for _, campaign := range campaigns {
			if !campaign.WeekStartDate.After(weekStart) && !campaign.WeekStartDate.Before(weekStart) {
				status.WeekPlannedQuantity += campaign.PlannedQuantity
				status.WeekScannedQuantity += campaign.ScannedQuantity
				status.WeekPlannedRunHours += campaign.PlannedRunHours
				if status.Unit == "" {
					status.Unit = campaign.Unit
				}
			}
		}

		if current != nil {
			status.Status = constants.MachineWorkStatusRunning
		}
		// Down last, and unconditionally: a broken machine with a released campaign is not producing, whatever the plan says.
		if downtime, ok := downByMachine[machine.ID]; ok {
			status.Downtime = downtime
			status.Status = constants.MachineWorkStatusDown
		}

		result.Machines = append(result.Machines, status)
	}

	return result, nil
}

// currentAndNext picks what a machine is on and what follows.
//
// Current is the earliest released campaign with batches still to scan. Released work that is fully scanned is finished, so it stops being current and the queue moves on — which is what makes a floor display advance on its own as the shift progresses.
func currentAndNext(campaigns []domain.MachineCampaign, weekStart time.Time) (*domain.MachineCampaign, *domain.MachineCampaign) {
	currentIdx := -1
	for i, campaign := range campaigns {
		isReleased := campaign.ProductionRunID != nil
		hasWorkLeft := campaign.ReleasedBatchCount == 0 || campaign.ScannedBatchCount < campaign.ReleasedBatchCount
		if isReleased && hasWorkLeft {
			currentIdx = i
			break
		}
	}

	// Nothing released is outstanding. The machine is between jobs, so the earliest campaign still ahead of it is what comes next rather than what it is on.
	if currentIdx == -1 {
		for i, campaign := range campaigns {
			if campaign.ScannedBatchCount == 0 && !campaign.WeekStartDate.Before(weekStart) {
				c := campaigns[i]
				return nil, &c
			}
		}
		return nil, nil
	}

	current := campaigns[currentIdx]
	if currentIdx+1 < len(campaigns) {
		next := campaigns[currentIdx+1]
		return &current, &next
	}
	return &current, nil
}
