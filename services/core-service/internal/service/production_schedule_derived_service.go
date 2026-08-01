package service

import (
	"context"

	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/scheduling"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/id"
	"github.com/augno/api/shared/safeconv"
	"github.com/augno/api/shared/tracing"
)

// deriveDepartmentWork explodes a version's constraint plan through the production-step graph and replaces that version's derived rows.
//
// Derived work is a pure function of the constraint plan, so it is regenerated wholesale rather than patched: a partial update could leave a department reading work for a campaign that no longer exists.
func (s *productionScheduleSvcImpl) deriveDepartmentWork(ctx context.Context, accountID, scheduleID string) *apierror.APIError {
	repo := s.repos.NewProductionScheduleRepo()

	schedule, apiErr := repo.Get(ctx, domain.GetProductionScheduleParams{
		AccountID:  accountID,
		ScheduleID: scheduleID,
	})
	if apiErr != nil {
		return apiErr
	}

	lines, apiErr := repo.ListLines(ctx, domain.ListProductionScheduleLinesParams{
		AccountID:  accountID,
		ScheduleID: scheduleID,
	})
	if apiErr != nil {
		return apiErr
	}

	graph, apiErr := repo.LoadStepGraph(ctx, accountID)
	if apiErr != nil {
		return apiErr
	}

	policies, apiErr := repo.ListItemPolicies(ctx, accountID, scheduleID)
	if apiErr != nil {
		return apiErr
	}
	skuByItemID := make(map[string]string, len(policies))
	for _, policy := range policies {
		skuByItemID[policy.ItemID] = policy.SKU
	}

	campaigns := make([]scheduling.ExplosionCampaign, 0, len(lines))
	for _, line := range lines {
		// A line with no production step has nothing downstream of it to derive from.
		if line.ProductionStepID == nil || *line.ProductionStepID == "" {
			continue
		}
		campaigns = append(campaigns, scheduling.ExplosionCampaign{
			LineID:    line.ID,
			ItemID:    line.ItemID,
			SKU:       skuByItemID[line.ItemID],
			MachineID: line.MachineID,
			WeekIndex: int(line.WeekIndex),
			Quantity:  line.PlannedQuantity,
			StepID:    *line.ProductionStepID,
		})
	}

	derived := scheduling.Explode(scheduling.ExplosionInput{
		Campaigns: campaigns,
		Edges:     graph.Edges,
		Steps:     graph.Steps,
	})

	rows := make([]*domain.ProductionScheduleDerivedLine, 0, len(derived))
	for _, line := range derived {
		derivedID, apiErr := id.GenID(id.ProductionScheduleDerivedLineIDPrefix, nil)
		if apiErr != nil {
			return apiErr
		}

		row := &domain.ProductionScheduleDerivedLine{
			ID:                   derivedID,
			ProductionScheduleID: scheduleID,
			SourceLineID:         line.SourceLineID,
			ProductionStepID:     line.ProductionStep,
			ItemID:               line.ItemID,
			WeekIndex:            safeconv.IntToInt32(line.WeekIndex),
			// The derived week can fall past the horizon: a step six weeks downstream of a week-12 campaign lands outside it. The date is still computed so the work list can show it rather than silently dropping the tail of the plan.
			WeekStartDate:  schedule.HorizonStartDate.AddDate(0, 0, line.WeekIndex*7),
			Quantity:       line.Quantity,
			ExplosionDepth: safeconv.IntToInt32(line.Depth),
			OffsetWeeks:    safeconv.IntToInt32(line.WeekIndex) - weekIndexOfSource(campaigns, line.SourceLineID),
			StatusCode:     domain.ScheduleLineStatusPlanned,
		}
		if line.DepartmentID != "" {
			department := line.DepartmentID
			row.DepartmentID = &department
		}
		rows = append(rows, row)
	}

	return repo.ReplaceDerivedLines(ctx, accountID, scheduleID, rows)
}

// weekIndexOfSource finds the constraint week a derived row followed from, so the stored offset is the accumulated lead time rather than an absolute week.
func weekIndexOfSource(campaigns []scheduling.ExplosionCampaign, lineID string) int32 {
	for _, campaign := range campaigns {
		if campaign.LineID == lineID {
			return safeconv.IntToInt32(campaign.WeekIndex)
		}
	}
	return 0
}

// ListProductionScheduleDerivedLines returns derived downstream department work.
func (s *productionScheduleSvcImpl) ListProductionScheduleDerivedLines(ctx context.Context, params domain.ListDerivedLinesParams) ([]*domain.ProductionScheduleDerivedLine, *apierror.APIError) {
	ctx, span := productionScheduleSvcTracer.Start(ctx, "service.production_schedule.list_derived_lines")
	defer span.End()

	identity, apiErr := s.readIdentity(ctx)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainProductionSchedules, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID
	return s.repos.NewProductionScheduleRepo().ListDerivedLines(ctx, params)
}
