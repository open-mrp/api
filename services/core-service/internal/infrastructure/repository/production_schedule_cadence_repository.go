package repository

import (
	"context"
	gosql "database/sql"
	"time"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/tracing"
)

func (r *productionScheduleRepoImpl) ListGenerationCadences(ctx context.Context) ([]domain.GenerationCadence, *apierror.APIError) {
	ctx, span := productionScheduleRepoTracer.Start(ctx, "repository.production_schedule.list_generation_cadences")
	defer span.End()

	rows, err := r.queries.ListAccountsWithGenerationCadence(ctx)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	out := make([]domain.GenerationCadence, len(rows))
	for i, row := range rows {
		cadence := domain.GenerationCadence{
			AccountID:   row.AccountID,
			Timezone:    row.GenerationTimezone,
			AutoPublish: row.AutoPublish,
			CreatedAt:   row.CreatedAt,
		}
		if row.GenerationCron.Valid {
			cadence.Cron = row.GenerationCron.String
		}
		if row.LastGeneratedAt.Valid {
			cadence.LastGeneratedAt = &row.LastGeneratedAt.Time
		}
		out[i] = cadence
	}
	return out, nil
}

func (r *productionScheduleRepoImpl) StampGenerationRun(ctx context.Context, accountID string, at time.Time) *apierror.APIError {
	ctx, span := productionScheduleRepoTracer.Start(ctx, "repository.production_schedule.stamp_generation_run")
	defer span.End()

	err := r.queries.StampGenerationRun(ctx, sqlc.StampGenerationRunParams{
		AccountID:       accountID,
		LastGeneratedAt: gosql.NullTime{Time: at, Valid: true},
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *productionScheduleRepoImpl) ReapStalledGenerations(ctx context.Context, before time.Time) *apierror.APIError {
	ctx, span := productionScheduleRepoTracer.Start(ctx, "repository.production_schedule.reap_stalled_generations")
	defer span.End()

	err := r.queries.ReapStalledGeneratingSchedules(ctx, before)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *productionScheduleRepoImpl) CreateGeneratingSchedule(ctx context.Context, params domain.CreateGeneratingScheduleParams) *apierror.APIError {
	ctx, span := productionScheduleRepoTracer.Start(ctx, "repository.production_schedule.create_generating")
	defer span.End()

	err := r.queries.CreateGeneratingProductionSchedule(ctx, sqlc.CreateGeneratingProductionScheduleParams{
		ID:               params.ID,
		AccountID:        params.AccountID,
		Version:          params.Version,
		Name:             dtNullString(params.Name),
		PlanningAsOf:     params.PlanningAsOf,
		HorizonStartDate: params.HorizonStartDate,
		HorizonEndDate:   params.HorizonEndDate,
		HorizonWeeks:     params.HorizonWeeks,
		FrozenWeeks:      params.FrozenWeeks,
		DemandBasisCode:  params.DemandBasisCode,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *productionScheduleRepoImpl) FillGeneratedSchedule(ctx context.Context, schedule *domain.ProductionSchedule) *apierror.APIError {
	ctx, span := productionScheduleRepoTracer.Start(ctx, "repository.production_schedule.fill_generated")
	defer span.End()

	err := r.queries.FillGeneratedProductionSchedule(ctx, sqlc.FillGeneratedProductionScheduleParams{
		AccountID:        schedule.AccountID,
		ID:               schedule.ID,
		Name:             dtNullString(schedule.Name),
		HorizonStartDate: schedule.HorizonStartDate,
		HorizonEndDate:   schedule.HorizonEndDate,
		HorizonWeeks:     schedule.HorizonWeeks,
		FrozenWeeks:      schedule.FrozenWeeks,
		DemandBasisCode:  schedule.DemandBasisCode,
		SolverVersion:    schedule.SolverVersion,
		SettingsSnapshot: schedule.SettingsSnapshot,
		Diagnostics:      schedule.Diagnostics,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *productionScheduleRepoImpl) FailGeneration(ctx context.Context, accountID, scheduleID, reason string) *apierror.APIError {
	ctx, span := productionScheduleRepoTracer.Start(ctx, "repository.production_schedule.fail_generation")
	defer span.End()

	err := r.queries.FailProductionScheduleGeneration(ctx, sqlc.FailProductionScheduleGenerationParams{
		AccountID:    accountID,
		ID:           scheduleID,
		ErrorMessage: gosql.NullString{String: reason, Valid: true},
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *productionScheduleRepoImpl) RefreshRegenerated(ctx context.Context, schedule *domain.ProductionSchedule) *apierror.APIError {
	ctx, span := productionScheduleRepoTracer.Start(ctx, "repository.production_schedule.refresh_regenerated")
	defer span.End()

	err := r.queries.RefreshRegeneratedSchedule(ctx, sqlc.RefreshRegeneratedScheduleParams{
		AccountID:        schedule.AccountID,
		ID:               schedule.ID,
		PlanningAsOf:     schedule.PlanningAsOf,
		HorizonStartDate: schedule.HorizonStartDate,
		HorizonEndDate:   schedule.HorizonEndDate,
		HorizonWeeks:     schedule.HorizonWeeks,
		FrozenWeeks:      schedule.FrozenWeeks,
		DemandBasisCode:  schedule.DemandBasisCode,
		SolverVersion:    schedule.SolverVersion,
		SettingsSnapshot: schedule.SettingsSnapshot,
		Diagnostics:      schedule.Diagnostics,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}
