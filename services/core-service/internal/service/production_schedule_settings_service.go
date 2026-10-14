package service

import (
	"context"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/audit"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/id"
	"github.com/open-mrp/api/shared/messaging"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/tracing"
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
		return apierror.NewFieldError("planning_horizon_weeks", apierror.CodeInvalidFormat, "The planning horizon must be between 1 and 104 weeks.")
	case s.FrozenWeeks < 0 || s.FrozenWeeks > s.PlanningHorizonWeeks:
		return apierror.NewFieldError("frozen_weeks", apierror.CodeInvalidFormat, "The frozen window cannot be longer than the horizon.")
	case s.ShiftsPerDay < 1:
		return apierror.NewFieldError("shifts_per_day", apierror.CodeInvalidFormat, "There must be at least one shift per day.")
	case s.HoursPerShift <= 0:
		return apierror.NewFieldError("hours_per_shift", apierror.CodeInvalidFormat, "A shift must be longer than zero hours.")
	case s.WorkDaysPerWeek < 1 || s.WorkDaysPerWeek > 7:
		return apierror.NewFieldError("work_days_per_week", apierror.CodeInvalidFormat, "Work days per week must be between 1 and 7.")
	case s.CapacityHeadroomPct <= 0 || s.CapacityHeadroomPct > 1:
		return apierror.NewFieldError("capacity_headroom_pct", apierror.CodeInvalidFormat, "Capacity headroom must be greater than 0 and at most 1.")
	case s.DefaultLotUnits <= 0:
		return apierror.NewFieldError("default_lot_units", apierror.CodeInvalidFormat, "The default lot size must be greater than zero.")
	case s.ChangeoverMinMinutes > s.ChangeoverMaxMinutes:
		return apierror.NewFieldError("changeover_min_minutes", apierror.CodeInvalidFormat, "The minimum changeover cannot exceed the maximum.")
	case s.MaxWeeksSupply <= 0:
		return apierror.NewFieldError("max_weeks_supply", apierror.CodeInvalidFormat, "Maximum weeks of supply must be greater than zero.")
	case s.WeekStartDay < 0 || s.WeekStartDay > 6:
		return apierror.NewFieldError("week_start_day", apierror.CodeInvalidFormat, "The week start day must be between 0 and 6.")
	// Zero is allowed — same-day shipping is a real commitment — but a negative would date every order's ship-by before the order itself.
	case s.DefaultCustomerLeadTimeDays < 0:
		return apierror.NewFieldError("default_customer_lead_time_days", apierror.CodeInvalidFormat, "The default customer lead time cannot be negative.")
	case s.DefaultFulfillmentPolicyCode != "" && !constants.FulfillmentPolicy(s.DefaultFulfillmentPolicyCode).IsValid():
		return apierror.NewFieldError("default_fulfillment_policy", apierror.CodeInvalidFormat, "Unknown fulfillment policy.")
	}

	// A cadence with an unparseable cron would fail silently on every tick, and the merchant would see nothing rather than an error.
	if s.IsEnabled {
		if s.GenerationCron == nil || *s.GenerationCron == "" {
			return apierror.NewFieldError("generation_cron", apierror.CodeInvalidFormat, "A generation schedule is required when the cadence is enabled.")
		}
		if _, err := cron.ParseStandard(*s.GenerationCron); err != nil {
			return apierror.NewFieldError("generation_cron", apierror.CodeInvalidFormat, "The generation schedule is not a valid cron expression.")
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
		// The changeover labor rate is cost data. A caller who cannot read it is shown null and so cannot send it back; the stored rate is kept rather than replaced by whatever such a caller sent.
		if before != nil && !identity.CanReadCosts() {
			settings.ChangeoverLaborRate = before.ChangeoverLaborRate
		}
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
		return nil, tracing.Trace(span, apierror.NewFieldError("scope_type", apierror.CodeInvalidFormat, "Unknown resource scope."))
	}

	settingID, apiErr := id.GenID(id.ProductionScheduleResourceSettingIDPrefix, nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID
	sameScope := func(setting *domain.ProductionScheduleResourceSetting) bool {
		return setting.ScopeCode == params.ScopeCode && setting.ScopeRefID == params.ScopeRefID
	}

	var result *domain.ProductionScheduleResourceSetting
	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *productionScheduleSvcImpl) *apierror.APIError {
		before, apiErr := txSvc.findResourceSetting(txCtx, params.AccountID, sameScope)
		if apiErr != nil {
			return apiErr
		}
		if apiErr := txSvc.repos.NewProductionScheduleRepo().UpsertResourceSetting(txCtx, settingID, params); apiErr != nil {
			return apiErr
		}
		after, apiErr := txSvc.findResourceSetting(txCtx, params.AccountID, sameScope)
		if apiErr != nil {
			return apiErr
		}
		if after == nil {
			return apierror.NewInternalError(nil, "Resource setting was not stored.")
		}
		result = after
		return publishSettingUpsert(txCtx, txSvc.repos.NewOutboxRepo(), constants.ObjectTypeProductionScheduleResourceSetting, after.ID, before, after)
	})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return result, nil
}

func (s *productionScheduleSvcImpl) findResourceSetting(ctx context.Context, accountID string, match func(*domain.ProductionScheduleResourceSetting) bool) (*domain.ProductionScheduleResourceSetting, *apierror.APIError) {
	settings, apiErr := s.repos.NewProductionScheduleRepo().ListResourceSettings(ctx, accountID)
	if apiErr != nil {
		return nil, apiErr
	}
	for _, setting := range settings {
		if match(setting) {
			return setting, nil
		}
	}
	return nil, nil
}

// publishSettingUpsert audits an upsert as a create when nothing was stored before, otherwise as an update carrying the diff.
func publishSettingUpsert[T any](ctx context.Context, outbox messaging.OutboxRepo, resourceType constants.ObjectType, resourceID string, before, after *T) *apierror.APIError {
	action := constants.AuditActionUpdate
	if before == nil {
		action = constants.AuditActionCreate
	}
	return audit.NewPublisher().Publish(ctx, outbox, audit.EventData{
		ServiceName:  domain.ServiceName,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Changes:      audit.ComputeChanges(before, after),
	})
}

func publishSettingDelete[T any](ctx context.Context, outbox messaging.OutboxRepo, resourceType constants.ObjectType, resourceID string, before *T) *apierror.APIError {
	return audit.NewPublisher().Publish(ctx, outbox, audit.EventData{
		ServiceName:  domain.ServiceName,
		Action:       constants.AuditActionDelete,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Changes:      audit.ComputeChanges(before, (*T)(nil)),
	})
}

// DeleteResourceSetting removes one per-resource override.
func (s *productionScheduleSvcImpl) DeleteResourceSetting(ctx context.Context, settingID string) *apierror.APIError {
	ctx, span := productionScheduleSvcTracer.Start(ctx, "service.production_schedule.delete_resource_setting")
	defer span.End()

	identity, apiErr := s.writeIdentity(ctx, types.ActionUpdate)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	accountID := identity.Target.AccountID

	return tracing.Trace(span, s.withTx(ctx, func(txCtx context.Context, txSvc *productionScheduleSvcImpl) *apierror.APIError {
		before, apiErr := txSvc.findResourceSetting(txCtx, accountID, func(setting *domain.ProductionScheduleResourceSetting) bool {
			return setting.ID == settingID
		})
		if apiErr != nil {
			return apiErr
		}
		if before == nil {
			return apierror.NewNotFoundError("Production schedule resource setting not found.")
		}
		if apiErr := txSvc.repos.NewProductionScheduleRepo().DeleteResourceSetting(txCtx, accountID, settingID); apiErr != nil {
			return apiErr
		}
		return publishSettingDelete(txCtx, txSvc.repos.NewOutboxRepo(), constants.ObjectTypeProductionScheduleResourceSetting, before.ID, before)
	}))
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
		return nil, tracing.Trace(span, apierror.NewNotFoundError("This item has no planning override."))
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
			return nil, tracing.Trace(span, apierror.NewFieldError("fulfillment_policy", apierror.CodeInvalidFormat, "Unknown fulfillment policy."))
		}
	}
	if params.LotMultipleUnits != nil && *params.LotMultipleUnits <= 0 {
		return nil, tracing.Trace(span, apierror.NewFieldError("lot_multiple_units", apierror.CodeInvalidFormat, "The lot multiple must be greater than zero."))
	}

	// A setting pointing at an item that does not exist would be invisible to the solver and unexplainable to whoever wrote it.
	items, apiErr := s.repos.NewItemRepo().GetByIDs(ctx, params.AccountID, []string{params.ItemID})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if len(items) == 0 {
		return nil, tracing.Trace(span, apierror.NewFieldError("item_id", apierror.CodeInvalidFormat, "Item not found."))
	}

	var setting *domain.ProductionScheduleItemPlanningSetting
	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *productionScheduleSvcImpl) *apierror.APIError {
		stored, apiErr := txSvc.writeItemSetting(txCtx, params)
		setting = stored
		return apiErr
	})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return setting, nil
}

// writeItemSetting upserts one item's override and audits the write; callers validate params and own the transaction.
func (s *productionScheduleSvcImpl) writeItemSetting(ctx context.Context, params domain.UpsertItemSettingParams) (*domain.ProductionScheduleItemPlanningSetting, *apierror.APIError) {
	repo := s.repos.NewProductionScheduleRepo()
	before, apiErr := repo.GetItemSetting(ctx, params.AccountID, params.ItemID)
	if apiErr != nil {
		return nil, apiErr
	}
	if apiErr := repo.UpsertItemSetting(ctx, params); apiErr != nil {
		return nil, apiErr
	}

	after, apiErr := repo.GetItemSetting(ctx, params.AccountID, params.ItemID)
	if apiErr != nil {
		return nil, apiErr
	}
	if after == nil {
		return nil, apierror.NewInvariantViolationError("The item setting vanished immediately after being written.")
	}
	if apiErr := publishSettingUpsert(ctx, s.repos.NewOutboxRepo(), constants.ObjectTypeProductionScheduleItemSetting, after.ID, before, after); apiErr != nil {
		return nil, apiErr
	}
	return after, nil
}

// DeleteItemSetting removes one item's override, returning it to the defaults.
func (s *productionScheduleSvcImpl) DeleteItemSetting(ctx context.Context, itemID string) *apierror.APIError {
	ctx, span := productionScheduleSvcTracer.Start(ctx, "service.production_schedule.delete_item_setting")
	defer span.End()

	identity, apiErr := s.writeIdentity(ctx, types.ActionUpdate)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	accountID := identity.Target.AccountID

	return tracing.Trace(span, s.withTx(ctx, func(txCtx context.Context, txSvc *productionScheduleSvcImpl) *apierror.APIError {
		repo := txSvc.repos.NewProductionScheduleRepo()
		before, apiErr := repo.GetItemSetting(txCtx, accountID, itemID)
		if apiErr != nil {
			return apiErr
		}
		// A delete that matched nothing is a 404: telling a caller their override is gone when it never existed hides a mistyped id.
		if before == nil {
			return apierror.NewNotFoundError("This item has no planning override.")
		}
		deleted, apiErr := repo.DeleteItemSetting(txCtx, accountID, itemID)
		if apiErr != nil {
			return apiErr
		}
		if !deleted {
			return apierror.NewNotFoundError("This item has no planning override.")
		}
		return publishSettingDelete(txCtx, txSvc.repos.NewOutboxRepo(), constants.ObjectTypeProductionScheduleItemSetting, before.ID, before)
	}))
}
