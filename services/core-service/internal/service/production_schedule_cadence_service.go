package service

import (
	"context"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/event"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/id"
	"github.com/augno/api/shared/safeconv"
	"github.com/augno/api/shared/tracing"
)

// EnqueueScheduledGeneration reserves a version and queues the solve, in one transaction.
//
// The placeholder row and the message that will fill it in commit together. Publishing first would let a message escape a transaction that then rolled back, and the consumer would solve into a row that does not exist; creating the row first without the message would leave a version stuck in `generating` with nothing coming to finish it.
func (s *productionScheduleSvcImpl) EnqueueScheduledGeneration(ctx context.Context, params domain.EnqueueGenerationParams) *apierror.APIError {
	ctx, span := productionScheduleSvcTracer.Start(ctx, "service.production_schedule.enqueue_scheduled_generation")
	defer span.End()

	scheduleID, apiErr := id.GenID(id.ProductionScheduleIDPrefix, nil)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *productionScheduleSvcImpl) *apierror.APIError {
		repo := txSvc.repos.NewProductionScheduleRepo()

		settings, apiErr := txSvc.loadEffectiveSettings(txCtx, params.AccountID)
		if apiErr != nil {
			return apiErr
		}

		version, apiErr := repo.NextVersion(txCtx, params.AccountID)
		if apiErr != nil {
			return apiErr
		}

		horizonStart := weekStart(params.PlanningAsOf)
		horizonEnd := horizonStart.AddDate(0, 0, settings.Settings.HorizonWeeks*7-1)

		if apiErr := repo.CreateGeneratingSchedule(txCtx, domain.CreateGeneratingScheduleParams{
			ID:               scheduleID,
			AccountID:        params.AccountID,
			Version:          version,
			PlanningAsOf:     params.PlanningAsOf,
			HorizonStartDate: horizonStart,
			HorizonEndDate:   horizonEnd,
			HorizonWeeks:     safeconv.IntToInt32(settings.Settings.HorizonWeeks),
			FrozenWeeks:      safeconv.IntToInt32(settings.Settings.FrozenWeeks),
			DemandBasisCode:  settings.DemandBasisCode,
		}); apiErr != nil {
			return apiErr
		}

		// The outbox publisher reads the repo factory from the context, so the transaction's factory has to be attached or the write lands outside it.
		outboxCtx := event.WithRepos(txCtx, txSvc.repos)
		return txSvc.enqueuer.EnqueueGeneration(outboxCtx, domain.EnqueueGenerationParams{
			AccountID:    params.AccountID,
			ScheduleID:   scheduleID,
			PlanningAsOf: params.PlanningAsOf,
			AutoPublish:  params.AutoPublish,
		})
	})
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

// RunScheduledGeneration solves into a row already created in `generating`.
//
// Called by the queue consumer, out of band from the cadence tick. A failure marks the version `failed` with the reason rather than leaving it in `generating`, so a merchant sees that the cadence ran and what went wrong instead of a silent gap.
func (s *productionScheduleSvcImpl) RunScheduledGeneration(ctx context.Context, params domain.RunScheduledGenerationParams) *apierror.APIError {
	ctx, span := productionScheduleSvcTracer.Start(ctx, "service.production_schedule.run_scheduled_generation")
	defer span.End()

	repo := s.repos.NewProductionScheduleRepo()

	schedule, apiErr := repo.Get(ctx, domain.GetProductionScheduleParams{
		AccountID:  params.AccountID,
		ScheduleID: params.ScheduleID,
	})
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	// Anything other than `generating` means this message duplicates one already handled. Redelivery is normal, and solving twice would give one version two plans.
	if schedule.StatusCode != domain.ScheduleStatusGenerating {
		return nil
	}

	if _, apiErr := s.persistPlan(ctx, persistPlanParams{
		AccountID:          params.AccountID,
		PlanningAsOf:       params.PlanningAsOf,
		SourceCode:         domain.ScheduleSourceScheduled,
		ExistingScheduleID: params.ScheduleID,
	}); apiErr != nil {
		// Best effort: if the status write also fails the reaper eventually catches it.
		if failErr := repo.FailGeneration(ctx, params.AccountID, params.ScheduleID, apiErr.PublicMessage); failErr != nil {
			return tracing.Trace(span, failErr)
		}
		return tracing.Trace(span, apiErr)
	}

	if params.AutoPublish {
		// The consumer's context carries no identity, so this goes through the identity-free publish core with the trusted account ID from the outbox message. A nil publishedByID matches a cadence run, which has no human actor.
		if _, publishErr := s.publishSchedule(ctx, params.AccountID, params.ScheduleID, nil); publishErr != nil {
			// The plan solved; only the publish failed. The draft stays so a planner can publish by hand rather than losing the whole run.
			return tracing.Trace(span, publishErr)
		}
	}

	return nil
}
