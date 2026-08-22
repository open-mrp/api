package repository

import (
	"context"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/services/core-service/internal/scheduling"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/tracing"
)

// LoadStepGraph returns the production-step graph plus the per-step metadata the explosion needs, with lead-time offsets already resolved from resource settings.
func (r *productionScheduleRepoImpl) LoadStepGraph(ctx context.Context, accountID string) (*domain.StepGraph, *apierror.APIError) {
	ctx, span := productionScheduleRepoTracer.Start(ctx, "repository.production_schedule.load_step_graph")
	defer span.End()

	edgeRows, err := r.queries.GetProductionStepGraph(ctx, accountID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	stepRows, err := r.queries.GetProductionStepsForExplosion(ctx, accountID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Per-step overrides carry the lead-time offset: how long after the constraint a department actually starts. Without them every downstream step would be planned in the same week as the constraint, which no real flow does.
	settingRows, err := r.queries.ListProductionScheduleResourceSettings(ctx, accountID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	offsetByStep := map[string]int{}
	for _, setting := range settingRows {
		if setting.ScopeCode != domain.ScheduleResourceScopeProductionStep {
			continue
		}
		offsetByStep[setting.ScopeRefID] = int(decimalToFloat64(setting.LeadTimeOffsetWeeks))
	}

	graph := &domain.StepGraph{
		Edges: make([]scheduling.StepEdge, 0, len(edgeRows)),
		Steps: make(map[string]scheduling.StepInfo, len(stepRows)),
	}
	for _, edge := range edgeRows {
		graph.Edges = append(graph.Edges, scheduling.StepEdge{
			UpstreamStepID:   edge.UpstreamStepID,
			DownstreamStepID: edge.DownstreamStepID,
		})
	}
	for _, step := range stepRows {
		info := scheduling.StepInfo{
			StepID: step.ID,
			Name:   step.Name,
			// Yield is not modelled per step yet, so no loss is assumed. Inventing a ratio would put a number on the plan that nobody configured.
			YieldRatio:          1,
			LeadTimeOffsetWeeks: offsetByStep[step.ID],
		}
		if step.DepartmentID.Valid {
			info.DepartmentID = step.DepartmentID.String
		}
		graph.Steps[step.ID] = info
	}

	return graph, nil
}

func (r *productionScheduleRepoImpl) ReplaceDerivedLines(ctx context.Context, accountID, scheduleID string, lines []*domain.ProductionScheduleDerivedLine) *apierror.APIError {
	ctx, span := productionScheduleRepoTracer.Start(ctx, "repository.production_schedule.replace_derived_lines")
	defer span.End()

	// Derived work is regenerated wholesale, never patched: it is a pure function of the constraint plan, so a partial update could leave rows describing a plan that no longer exists.
	if err := r.queries.DeleteProductionScheduleDerivedLines(ctx, sqlc.DeleteProductionScheduleDerivedLinesParams{
		AccountID:            accountID,
		ProductionScheduleID: scheduleID,
	}); err != nil {
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
	}

	for _, line := range lines {
		if err := r.queries.CreateProductionScheduleDerivedLine(ctx, sqlc.CreateProductionScheduleDerivedLineParams{
			ID:                   line.ID,
			AccountID:            accountID,
			ProductionScheduleID: scheduleID,
			SourceLineID:         line.SourceLineID,
			ProductionStepID:     line.ProductionStepID,
			DepartmentID:         dtNullString(line.DepartmentID),
			ItemID:               line.ItemID,
			WeekIndex:            line.WeekIndex,
			WeekStartDate:        line.WeekStartDate,
			Quantity:             floatToDecimalString(line.Quantity),
			PlannedUnitID:        dtNullString(line.PlannedUnitID),
			ExplosionDepth:       line.ExplosionDepth,
			OffsetWeeks:          line.OffsetWeeks,
			StatusCode:           line.StatusCode,
		}); err != nil {
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return tracing.Trace(span, apiErr)
			}
		}
	}

	return nil
}

func (r *productionScheduleRepoImpl) ListDerivedLines(ctx context.Context, params domain.ListDerivedLinesParams) ([]*domain.ProductionScheduleDerivedLine, *apierror.APIError) {
	ctx, span := productionScheduleRepoTracer.Start(ctx, "repository.production_schedule.list_derived_lines")
	defer span.End()

	rows, err := r.queries.ListProductionScheduleDerivedLines(ctx, sqlc.ListProductionScheduleDerivedLinesParams{
		AccountID:               params.AccountID,
		ProductionScheduleID:    params.ScheduleID,
		IncludeDepartmentFilter: len(params.DepartmentIDs) > 0,
		DepartmentIds:           dtNullStrings(params.DepartmentIDs),
		WeekIndex:               psNullInt32(params.WeekIndex),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	out := make([]*domain.ProductionScheduleDerivedLine, len(rows))
	for i, row := range rows {
		line := &domain.ProductionScheduleDerivedLine{
			ID:                   row.ID,
			ProductionScheduleID: row.ProductionScheduleID,
			SourceLineID:         row.SourceLineID,
			ProductionStepID:     row.ProductionStepID,
			ItemID:               row.ItemID,
			WeekIndex:            row.WeekIndex,
			WeekStartDate:        row.WeekStartDate,
			Quantity:             decimalToFloat64(row.Quantity),
			ExplosionDepth:       row.ExplosionDepth,
			OffsetWeeks:          row.OffsetWeeks,
			StatusCode:           row.StatusCode,
			CreatedAt:            row.CreatedAt,
			UpdatedAt:            row.UpdatedAt,
		}
		if row.DepartmentID.Valid {
			line.DepartmentID = &row.DepartmentID.String
		}
		if row.PlannedUnitID.Valid {
			line.PlannedUnitID = &row.PlannedUnitID.String
		}
		out[i] = line
	}
	return out, nil
}
