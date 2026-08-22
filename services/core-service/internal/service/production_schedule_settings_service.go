package service

import (
	"context"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/audit"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/id"
	"github.com/open-mrp/api/shared/tracing"
	"github.com/robfig/cron/v3"
)

// GetProductionScheduleSettings returns the merchant's planning assumptions, falling back to code defaults when nothing has been saved.
func (s *productionScheduleSvcImpl) GetProductionScheduleSettings(ctx context.Context) (*domain.ProductionScheduleSettings, *apierror.APIError) {
	ctx, span := productionScheduleSvcTracer.Start(ctx, "service.production_schedule.get_settings")
	defer span.End()

	identity, apiErr := s.readIdentity(ctx)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainProductionSchedules, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return s.repos.NewProductionScheduleRepo().GetSettings(ctx, identity.Target.AccountID)
}

// validateSettings rejects assumptions that would make the solver produce nonsense rather than fail. A zero-hour shift or a negative headroom does not error anywhere in the solve; it silently yields a plan with no capacity, which reads as "nothing to do".
func validateSettings(s domain.ProductionScheduleSettings) *apierror.APIError {
	switch {
	case s.PlanningHorizonWeeks < 1 || s.PlanningHorizonWeeks > 104:
		return apierror.NewValidationErrorWithParam("The planning horizon must be between 1 and 104 weeks.", "planning_horizon_weeks")
	case s.FrozenWeeks < 0 || s.FrozenWeeks > s.PlanningHorizonWeeks:
		return apierror.NewValidationErrorWithParam("The frozen window cannot be longer than the horizon.", "frozen_weeks")
	case s.ShiftsPerDay < 1:
		return apierror.NewValidationErrorWithParam("There must be at least one shift per day.", "shifts_per_day")
	case s.HoursPerShift <= 0:
		return apierror.NewValidationErrorWithParam("A shift must be longer than zero hours.", "hours_per_shift")
	case s.WorkDaysPerWeek < 1 || s.WorkDaysPerWeek > 7:
		return apierror.NewValidationErrorWithParam("Work days per week must be between 1 and 7.", "work_days_per_week")
	case s.CapacityHeadroomPct <= 0 || s.CapacityHeadroomPct > 1:
		return apierror.NewValidationErrorWithParam("Capacity headroom must be greater than 0 and at most 1.", "capacity_headroom_pct")
	case s.DefaultLotUnits <= 0:
		return apierror.NewValidationErrorWithParam("The default lot size must be greater than zero.", "default_lot_units")
	case s.ChangeoverMinMinutes > s.ChangeoverMaxMinutes:
		return apierror.NewValidationErrorWithParam("The minimum changeover cannot exceed the maximum.", "changeover_min_minutes")
	case s.MaxWeeksSupply <= 0:
		return apierror.NewValidationErrorWithParam("Maximum weeks of supply must be greater than zero.", "max_weeks_supply")
	case s.WeekStartDay < 0 || s.WeekStartDay > 6:
		return apierror.NewValidationErrorWithParam("The week start day must be between 0 and 6.", "week_start_day")
	// Zero is allowed — same-day shipping is a real commitment — but a negative would date every order's ship-by before the order itself.
	case s.DefaultCustomerLeadTimeDays < 0:
		return apierror.NewValidationErrorWithParam("The default customer lead time cannot be negative.", "default_customer_lead_time_days")
	case s.DefaultFulfillmentPolicyCode != "" && !constants.FulfillmentPolicy(s.DefaultFulfillmentPolicyCode).IsValid():
		return apierror.NewValidationErrorWithParam("Unknown fulfillment policy.", "default_fulfillment_policy")
	}

	// A cadence with an unparseable cron would fail silently on every tick, and the merchant would see nothing rather than an error.
	if s.IsEnabled {
		if s.GenerationCron == nil || *s.GenerationCron == "" {
			return apierror.NewValidationErrorWithParam("A generation schedule is required when the cadence is enabled.", "generation_cron")
		}
		if _, err := cron.ParseStandard(*s.GenerationCron); err != nil {
			return apierror.NewValidationErrorWithParam("The generation schedule is not a valid cron expression.", "generation_cron")
		}
	}

	return nil
}

// UpdateProductionScheduleSettings replaces the merchant's planning assumptions.
func (s *productionScheduleSvcImpl) UpdateProductionScheduleSettings(ctx context.Context, params domain.UpdateProductionScheduleSettingsParams) (*domain.ProductionScheduleSettings, *apierror.APIError) {
	ctx, span := productionScheduleSvcTracer.Start(ctx, "service.production_schedule.update_settings")
	defer span.End()

	identity, apiErr := s.writeIdentity(ctx, types.ActionUpdate)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	accountID := identity.Target.AccountID

	if apiErr := validateSettings(params.Settings); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	var result *domain.ProductionScheduleSettings
	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *productionScheduleSvcImpl) *apierror.APIError {
		repo := txSvc.repos.NewProductionScheduleRepo()

		before, apiErr := repo.GetSettings(txCtx, accountID)
		if apiErr != nil {
			return apiErr
		}

		settings := params.Settings
		settings.AccountID = accountID
		if apiErr := repo.UpsertSettings(txCtx, &settings); apiErr != nil {
			return apiErr
		}

		after, apiErr := repo.GetSettings(txCtx, accountID)
		if apiErr != nil {
			return apiErr
		}
		result = after

		// Settings change what every future plan means, so a change to them audits even though no schedule moved.
		return audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionUpdate,
			ResourceType: constants.ObjectTypeProductionScheduleSettings,
			ResourceID:   accountID,
			Changes:      audit.ComputeChanges(before, after),
		})
	})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return result, nil
}

// ListResourceSettings returns per-machine, per-department and per-step overrides.
func (s *productionScheduleSvcImpl) ListResourceSettings(ctx context.Context) ([]*domain.ProductionScheduleResourceSetting, *apierror.APIError) {
	ctx, span := productionScheduleSvcTracer.Start(ctx, "service.production_schedule.list_resource_settings")
	defer span.End()

	identity, apiErr := s.readIdentity(ctx)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainProductionSchedules, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return s.repos.NewProductionScheduleRepo().ListResourceSettings(ctx, identity.Target.AccountID)
}

// UpsertResourceSetting writes one per-resource override.
func (s *productionScheduleSvcImpl) UpsertResourceSetting(ctx context.Context, params domain.UpsertResourceSettingParams) (*domain.ProductionScheduleResourceSetting, *apierror.APIError) {
	ctx, span := productionScheduleSvcTracer.Start(ctx, "service.production_schedule.upsert_resource_setting")
	defer span.End()

	identity, apiErr := s.writeIdentity(ctx, types.ActionUpdate)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	switch params.ScopeCode {
	case domain.ScheduleResourceScopeMachine, domain.ScheduleResourceScopeDepartment, domain.ScheduleResourceScopeProductionStep:
	default:
		return nil, tracing.Trace(span, apierror.NewValidationErrorWithParam("Unknown resource scope.", "scope_type"))
	}

	settingID, apiErr := id.GenID(id.ProductionScheduleResourceSettingIDPrefix, nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID
	repo := s.repos.NewProductionScheduleRepo()
	if apiErr := repo.UpsertResourceSetting(ctx, settingID, params); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	settings, apiErr := repo.ListResourceSettings(ctx, params.AccountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	for _, setting := range settings {
		if setting.ScopeCode == params.ScopeCode && setting.ScopeRefID == params.ScopeRefID {
			return setting, nil
		}
	}
	return nil, tracing.Trace(span, apierror.NewInternalError(nil, "Resource setting was not stored."))
}

// DeleteResourceSetting removes one per-resource override.
func (s *productionScheduleSvcImpl) DeleteResourceSetting(ctx context.Context, settingID string) *apierror.APIError {
	ctx, span := productionScheduleSvcTracer.Start(ctx, "service.production_schedule.delete_resource_setting")
	defer span.End()

	identity, apiErr := s.writeIdentity(ctx, types.ActionUpdate)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return s.repos.NewProductionScheduleRepo().DeleteResourceSetting(ctx, identity.Target.AccountID, settingID)
}

// ListItemSettings returns every per-item planning override in the account.
func (s *productionScheduleSvcImpl) ListItemSettings(ctx context.Context) ([]*domain.ProductionScheduleItemPlanningSetting, *apierror.APIError) {
	ctx, span := productionScheduleSvcTracer.Start(ctx, "service.production_schedule.list_item_settings")
	defer span.End()

	identity, apiErr := s.readIdentity(ctx)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainProductionSchedules, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return s.repos.NewProductionScheduleRepo().ListItemSettings(ctx, identity.Target.AccountID)
}

// GetItemSetting returns one item's override.
func (s *productionScheduleSvcImpl) GetItemSetting(ctx context.Context, itemID string) (*domain.ProductionScheduleItemPlanningSetting, *apierror.APIError) {
	ctx, span := productionScheduleSvcTracer.Start(ctx, "service.production_schedule.get_item_setting")
	defer span.End()

	identity, apiErr := s.readIdentity(ctx)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainProductionSchedules, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	setting, apiErr := s.repos.NewProductionScheduleRepo().GetItemSetting(ctx, identity.Target.AccountID, itemID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if setting == nil {
		return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("This item has no planning override."))
	}
	return setting, nil
}

// UpsertItemSetting writes one item's planning override.
func (s *productionScheduleSvcImpl) UpsertItemSetting(ctx context.Context, params domain.UpsertItemSettingParams) (*domain.ProductionScheduleItemPlanningSetting, *apierror.APIError) {
	ctx, span := productionScheduleSvcTracer.Start(ctx, "service.production_schedule.upsert_item_setting")
	defer span.End()

	identity, apiErr := s.writeIdentity(ctx, types.ActionUpdate)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	params.AccountID = identity.Target.AccountID

	// An unknown policy would be stored and then silently ignored by the solver, leaving a merchant looking at a setting that does nothing.
	if params.FulfillmentPolicyCode != nil {
		if !constants.FulfillmentPolicy(*params.FulfillmentPolicyCode).IsValid() {
			return nil, tracing.Trace(span, apierror.NewValidationErrorWithParam("Unknown fulfillment policy.", "fulfillment_policy"))
		}
	}
	if params.LotMultipleUnits != nil && *params.LotMultipleUnits <= 0 {
		return nil, tracing.Trace(span, apierror.NewValidationErrorWithParam("The lot multiple must be greater than zero.", "lot_multiple_units"))
	}

	// A setting pointing at an item that does not exist would be invisible to the solver and unexplainable to whoever wrote it.
	items, apiErr := s.repos.NewItemRepo().GetByIDs(ctx, params.AccountID, []string{params.ItemID})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if len(items) == 0 {
		return nil, tracing.Trace(span, apierror.NewValidationErrorWithParam("Item not found.", "item_id"))
	}

	repo := s.repos.NewProductionScheduleRepo()
	if apiErr := repo.UpsertItemSetting(ctx, params); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	setting, apiErr := repo.GetItemSetting(ctx, params.AccountID, params.ItemID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if setting == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("The item setting vanished immediately after being written."))
	}
	return setting, nil
}

// DeleteItemSetting removes one item's override, returning it to the defaults.
func (s *productionScheduleSvcImpl) DeleteItemSetting(ctx context.Context, itemID string) *apierror.APIError {
	ctx, span := productionScheduleSvcTracer.Start(ctx, "service.production_schedule.delete_item_setting")
	defer span.End()

	identity, apiErr := s.writeIdentity(ctx, types.ActionUpdate)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	// A delete that matched nothing is a 404: telling a caller their override is gone when it never existed hides a mistyped id.
	deleted, apiErr := s.repos.NewProductionScheduleRepo().DeleteItemSetting(ctx, identity.Target.AccountID, itemID)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if !deleted {
		return tracing.Trace(span, apierror.NewResourceNotFoundError("This item has no planning override."))
	}
	return nil
}
