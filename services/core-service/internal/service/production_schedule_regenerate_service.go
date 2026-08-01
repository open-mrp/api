package service

import (
	"context"
	"encoding/json"
	"sort"
	"time"

	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/scheduling"
	"github.com/augno/api/shared/appctx"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/safeconv"
	"github.com/augno/api/shared/tracing"
)

// campaignKey identifies one campaign across two versions of the same plan.
//
// A campaign is a machine × item × week, so that tuple is what makes "the same campaign, in a different quantity" distinguishable from "a different campaign".
type campaignKey struct {
	ItemID    string
	MachineID string
	WeekIndex int32
}

// loadRegenerateTarget reads the version a regenerate is about to act on and refuses the ones where re-solving in place would rewrite rather than replan.
func (s *productionScheduleSvcImpl) loadRegenerateTarget(
	ctx context.Context,
	accountID, scheduleID string,
) (*domain.ProductionSchedule, *apierror.APIError) {
	schedule, apiErr := s.repos.NewProductionScheduleRepo().Get(ctx, domain.GetProductionScheduleParams{
		AccountID:  accountID,
		ScheduleID: scheduleID,
	})
	if apiErr != nil {
		return nil, apiErr
	}

	// Only a draft is regenerated. A published version is a commitment the floor is already working to, and a superseded or archived one is history — re-solving either in place would change what a week was measured against after the fact. The way to replan against a published version is to generate a new one, which supersedes it at publish.
	if schedule.StatusCode != domain.ScheduleStatusDraft {
		return nil, apierror.NewValidationError(
			"Only a draft schedule can be regenerated. Generate a new version instead; publishing it supersedes the current one.")
	}

	return schedule, nil
}

// solveForRegenerate re-solves using the version's own horizon and basis unless the caller overrides them.
//
// The planning instant is NOT reused: a regenerate answers "what would the solver say now", and replaying the draft's original instant would silently answer a different question — demand overrides created since then would be filtered out as not-yet-effective, and the horizon would stay anchored to the day the draft was first generated no matter how stale that is.
func (s *productionScheduleSvcImpl) solveForRegenerate(
	ctx context.Context,
	accountID string,
	schedule *domain.ProductionSchedule,
	params domain.RegenerateProductionScheduleParams,
) (*scheduling.SolverOutput, *domain.EffectiveScheduleSettings, time.Time, *apierror.APIError) {
	planningAsOf := time.Now().UTC()
	if params.PlanningAsOf != nil && !params.PlanningAsOf.IsZero() {
		planningAsOf = *params.PlanningAsOf
	}

	horizonWeeks := int(schedule.HorizonWeeks)
	if params.HorizonWeeks > 0 {
		horizonWeeks = params.HorizonWeeks
	}

	demandBasis := schedule.DemandBasisCode
	if params.DemandBasis != "" {
		demandBasis = params.DemandBasis
	}

	// When hand edits are being kept, the solver plans AROUND them: a hand-added campaign's stock and machine time are facts the rest of the plan responds to, and a trimmed one leaves a gap the solver refills. Without pinning, the fresh solve would quietly re-plan as if the edits did not exist.
	var pinned []scheduling.PinnedCampaign
	if resolveMergeMode(params.MergeMode) == domain.ScheduleMergeModePreserveManual {
		var apiErr *apierror.APIError
		pinned, apiErr = s.loadManualPins(ctx, accountID, schedule.ID, planningAsOf, horizonWeeks)
		if apiErr != nil {
			return nil, nil, time.Time{}, apiErr
		}
	}

	output, effective, apiErr := s.solveFor(ctx, accountID, planningAsOf, horizonWeeks, demandBasis, pinned)
	if apiErr != nil {
		return nil, nil, time.Time{}, apiErr
	}
	return output, effective, planningAsOf, nil
}

// resolveMergeMode applies the default: hand edits are preserved unless the caller explicitly asks to replace them.
func resolveMergeMode(mode string) string {
	if mode == "" {
		return domain.ScheduleMergeModePreserveManual
	}
	return mode
}

// scheduleLineWeekIndex places a stored line on the CURRENT horizon grid by its absolute week date. The stored week_index is relative to the horizon the line was written under, which a re-anchored regenerate may have moved.
func scheduleLineWeekIndex(weekStartDate, horizonStart time.Time) int {
	return int(weekStartDate.Sub(horizonStart).Hours() / (24 * 7))
}

// loadManualPins turns the draft's kept hand edits into solver pins on the fresh horizon grid.
//
// Completed and cancelled lines are not pinned: a completed campaign's output is already counted in on-hand stock, and a cancelled one will never arrive. Lines whose week falls outside the new horizon cannot constrain it.
func (s *productionScheduleSvcImpl) loadManualPins(
	ctx context.Context,
	accountID, scheduleID string,
	planningAsOf time.Time,
	horizonWeeks int,
) ([]scheduling.PinnedCampaign, *apierror.APIError) {
	weekStartDay := 1
	settingsRow, apiErr := s.repos.NewProductionScheduleInputRepo().GetAccountScheduleSettings(ctx, accountID)
	if apiErr != nil {
		return nil, apiErr
	}
	if settingsRow != nil {
		weekStartDay = settingsRow.WeekStartDay
	}
	horizonStart := scheduleWeekStart(planningAsOf, weekStartDay)

	lines, apiErr := s.repos.NewProductionScheduleRepo().ListLines(ctx, domain.ListProductionScheduleLinesParams{
		AccountID:  accountID,
		ScheduleID: scheduleID,
	})
	if apiErr != nil {
		return nil, apiErr
	}

	var pinned []scheduling.PinnedCampaign
	for _, line := range lines {
		if line.SourceCode != domain.ScheduleLineSourceManual {
			continue
		}
		if line.StatusCode == domain.ScheduleLineStatusComplete || line.StatusCode == domain.ScheduleLineStatusCancelled {
			continue
		}
		weekIndex := scheduleLineWeekIndex(line.WeekStartDate, horizonStart)
		if weekIndex < 0 || weekIndex >= horizonWeeks {
			continue
		}
		pinned = append(pinned, scheduling.PinnedCampaign{
			ItemID:    line.ItemID,
			MachineID: line.MachineID,
			WeekIndex: weekIndex,
			Units:     line.PlannedQuantity,
		})
	}
	return pinned, nil
}

// diffCampaigns compares the stored lines against a fresh solve.
func diffCampaigns(
	existing []*domain.ProductionScheduleLine,
	campaigns []scheduling.Campaign,
	skuByItemID map[string]string,
	horizonStart time.Time,
	preserveManual bool,
) []domain.ScheduleDiffLine {
	currentByKey := make(map[campaignKey]*domain.ProductionScheduleLine, len(existing))
	for _, line := range existing {
		// Stored lines are placed on the fresh solve's grid by absolute date, since a re-anchored regenerate may have moved the horizon their week_index was relative to.
		currentByKey[campaignKey{ItemID: line.ItemID, MachineID: line.MachineID, WeekIndex: safeconv.IntToInt32(scheduleLineWeekIndex(line.WeekStartDate, horizonStart))}] = line
	}

	proposedByKey := make(map[campaignKey]float64, len(campaigns))
	for _, c := range campaigns {
		key := campaignKey{ItemID: c.ItemID, MachineID: c.MachineID, WeekIndex: safeconv.IntToInt32(c.WeekIndex)}
		// A solve can place two campaigns for the same item, machine and week; they are one campaign as far as the diff is concerned.
		proposedByKey[key] += c.Units
	}

	// Kept hand edits are pinned during the solve, so the solver never re-proposes them; without this they would read as "removed" when they are in fact staying exactly as they are.
	if preserveManual {
		for key, line := range currentByKey {
			if line.SourceCode == domain.ScheduleLineSourceManual {
				proposedByKey[key] = line.PlannedQuantity
			}
		}
	}

	seen := make(map[campaignKey]bool, len(currentByKey)+len(proposedByKey))
	out := make([]domain.ScheduleDiffLine, 0, len(currentByKey)+len(proposedByKey))

	appendLine := func(key campaignKey) {
		if seen[key] {
			return
		}
		seen[key] = true

		line, hasCurrent := currentByKey[key]
		proposed, hasProposed := proposedByKey[key]

		diff := domain.ScheduleDiffLine{
			ItemID:    key.ItemID,
			SKU:       skuByItemID[key.ItemID],
			MachineID: key.MachineID,
			WeekIndex: key.WeekIndex,
		}
		if hasCurrent {
			diff.CurrentQuantity = line.PlannedQuantity
			diff.CurrentIsManual = line.SourceCode == domain.ScheduleLineSourceManual
		}
		if hasProposed {
			diff.ProposedQuantity = proposed
		}

		switch {
		case !hasCurrent:
			diff.ChangeCode = domain.ScheduleDiffAdded
		case !hasProposed:
			diff.ChangeCode = domain.ScheduleDiffRemoved
		case diff.CurrentQuantity != diff.ProposedQuantity:
			diff.ChangeCode = domain.ScheduleDiffChanged
		default:
			diff.ChangeCode = domain.ScheduleDiffUnchanged
		}

		out = append(out, diff)
	}

	for key := range currentByKey {
		appendLine(key)
	}
	for key := range proposedByKey {
		appendLine(key)
	}

	// Go randomizes map iteration, so the diff is sorted or the same regenerate would present its changes in a different order every time it is previewed.
	sort.Slice(out, func(i, j int) bool {
		if out[i].WeekIndex != out[j].WeekIndex {
			return out[i].WeekIndex < out[j].WeekIndex
		}
		if out[i].SKU != out[j].SKU {
			return out[i].SKU < out[j].SKU
		}
		if out[i].ItemID != out[j].ItemID {
			return out[i].ItemID < out[j].ItemID
		}
		return out[i].MachineID < out[j].MachineID
	})

	return out
}

// summarizePreview counts what the diff means for each merge mode.
func summarizePreview(lines []domain.ScheduleDiffLine) domain.ScheduleRegeneratePreview {
	preview := domain.ScheduleRegeneratePreview{Lines: lines}

	for _, line := range lines {
		switch line.ChangeCode {
		case domain.ScheduleDiffAdded:
			preview.AddedCount++
		case domain.ScheduleDiffRemoved:
			preview.RemovedCount++
		case domain.ScheduleDiffChanged:
			preview.ChangedCount++
		}

		if !line.CurrentIsManual {
			continue
		}
		preview.ManualLineCount++
		// A hand edit is destroyed by replace_all whenever the fresh solve disagrees with it — either by not wanting the campaign at all, or by wanting a different quantity. A manual line the solver happens to agree with survives either way.
		if line.ChangeCode == domain.ScheduleDiffRemoved || line.ChangeCode == domain.ScheduleDiffChanged {
			preview.DiscardedManualCount++
		}
	}

	return preview
}

// PreviewRegenerateProductionSchedule says what a regenerate would change, without changing anything.
//
// A regenerate that silently eats hand-work is abandoned within two cycles, so the destructive mode has to be able to state its cost as a number before it runs.
func (s *productionScheduleSvcImpl) PreviewRegenerateProductionSchedule(
	ctx context.Context,
	params domain.RegenerateProductionScheduleParams,
) (*domain.ScheduleRegeneratePreview, *apierror.APIError) {
	ctx, span := productionScheduleSvcTracer.Start(ctx, "service.production_schedule.preview_regenerate")
	defer span.End()

	identity, apiErr := s.readIdentity(ctx)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainProductionSchedules, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	accountID := identity.Target.AccountID

	schedule, apiErr := s.loadRegenerateTarget(ctx, accountID, params.ScheduleID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	output, effective, planningAsOf, apiErr := s.solveForRegenerate(ctx, accountID, schedule, params)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	existing, apiErr := s.repos.NewProductionScheduleRepo().ListLines(ctx, domain.ListProductionScheduleLinesParams{
		AccountID:  accountID,
		ScheduleID: params.ScheduleID,
	})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	skuByItemID := make(map[string]string, len(output.Policies))
	for _, policy := range output.Policies {
		skuByItemID[policy.ItemID] = policy.SKU
	}

	horizonStart := scheduleWeekStart(planningAsOf, effective.Settings.WeekStartDay)
	preserveManual := resolveMergeMode(params.MergeMode) == domain.ScheduleMergeModePreserveManual
	preview := summarizePreview(diffCampaigns(existing, output.Campaigns, skuByItemID, horizonStart, preserveManual))
	preview.ScheduleID = params.ScheduleID
	preview.SolverVersion = output.SolverVersion
	preview.PlanningAsOf = planningAsOf

	return &preview, nil
}

// RegenerateProductionSchedule re-solves a draft in place, keeping its version number.
//
// The version number is kept deliberately: a regenerate answers "solve this again", and minting a new version for every re-solve would fill the list with drafts nobody asked for and make the version number meaningless as a count of plans that were actually considered.
func (s *productionScheduleSvcImpl) RegenerateProductionSchedule(
	ctx context.Context,
	params domain.RegenerateProductionScheduleParams,
) (*domain.ProductionSchedule, *apierror.APIError) {
	ctx, span := productionScheduleSvcTracer.Start(ctx, "service.production_schedule.regenerate")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainProductionSchedules, types.ActionUpdate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	mergeMode := params.MergeMode
	if mergeMode == "" {
		// Preserving hand edits is the default because the alternative destroys work silently, and a caller that means to discard can say so.
		mergeMode = domain.ScheduleMergeModePreserveManual
	}
	if mergeMode != domain.ScheduleMergeModePreserveManual && mergeMode != domain.ScheduleMergeModeReplaceAll {
		return nil, tracing.Trace(span, apierror.NewValidationError("Unknown merge mode."))
	}

	accountID := identity.Target.AccountID

	schedule, apiErr := s.loadRegenerateTarget(ctx, accountID, params.ScheduleID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	output, effective, planningAsOf, apiErr := s.solveForRegenerate(ctx, accountID, schedule, params)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	var actorID *string
	if identity.Actor != nil {
		actorID = &identity.Actor.ID
	}

	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *productionScheduleSvcImpl) *apierror.APIError {
		repo := txSvc.repos.NewProductionScheduleRepo()

		existing, apiErr := repo.ListLines(txCtx, domain.ListProductionScheduleLinesParams{
			AccountID:  accountID,
			ScheduleID: params.ScheduleID,
		})
		if apiErr != nil {
			return apiErr
		}

		// A re-anchored regenerate can move the horizon start, so every kept line is placed on the NEW grid by its absolute week date — both for the skip keys the fresh solve honors and for the stored week_index, which release-by-week reads.
		horizonStart := scheduleWeekStart(planningAsOf, effective.Settings.WeekStartDay)

		keptManual := map[campaignKey]bool{}
		for _, line := range existing {
			isManual := line.SourceCode == domain.ScheduleLineSourceManual
			newWeekIndex := safeconv.IntToInt32(scheduleLineWeekIndex(line.WeekStartDate, horizonStart))
			key := campaignKey{ItemID: line.ItemID, MachineID: line.MachineID, WeekIndex: newWeekIndex}

			if isManual && mergeMode == domain.ScheduleMergeModePreserveManual {
				keptManual[key] = true
				if newWeekIndex != line.WeekIndex {
					reindexed := newWeekIndex
					if _, apiErr := repo.UpdateLine(txCtx, domain.UpdateLineRepoParams{
						AccountID: accountID,
						LineID:    line.ID,
						WeekIndex: &reindexed,
					}); apiErr != nil {
						return apiErr
					}
				}
				continue
			}

			// Every hand edit that is about to be destroyed is written to the deviation log first. "Where did my change go?" has to be answerable afterwards, and the alternative is a planner who stops trusting the plan.
			if isManual {
				actor := ""
				if actorID != nil {
					actor = *actorID
				}
				if apiErr := txSvc.recordDeviation(txCtx, schedule, snapshotLine(line), nil,
					line.WeekStartDate, nil, nil, actor); apiErr != nil {
					return apiErr
				}
			}

			if apiErr := repo.DeleteLine(txCtx, accountID, line.ID); apiErr != nil {
				return apiErr
			}
		}

		// The header carries the fresh solve's own metadata: a plan whose diagnostics and settings snapshot describe the previous solve cannot explain itself.
		if apiErr := txSvc.refreshRegeneratedHeader(txCtx, accountID, schedule, output, effective, planningAsOf); apiErr != nil {
			return apiErr
		}

		starved := map[string]bool{}
		for _, sku := range output.Diagnostics.CapacityStarvedSKUs {
			starved[sku] = true
		}
		capped := map[string]bool{}
		for _, sku := range output.Diagnostics.EOQCappedSKUs {
			capped[sku] = true
		}

		return txSvc.writeSolvedPlan(txCtx, writeSolvedPlanParams{
			AccountID:    accountID,
			ScheduleID:   params.ScheduleID,
			HorizonStart: scheduleWeekStart(planningAsOf, effective.Settings.WeekStartDay),
			Output:       output,
			Starved:      starved,
			Capped:       capped,
			SkipKeys:     keptManual,
		})
	})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	stored, apiErr := s.repos.NewProductionScheduleRepo().Get(ctx, domain.GetProductionScheduleParams{
		AccountID:  accountID,
		ScheduleID: params.ScheduleID,
	})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return stored, nil
}

// refreshRegeneratedHeader re-stamps the version with the metadata of the solve that just replaced its lines.
func (s *productionScheduleSvcImpl) refreshRegeneratedHeader(
	ctx context.Context,
	accountID string,
	schedule *domain.ProductionSchedule,
	output *scheduling.SolverOutput,
	effective *domain.EffectiveScheduleSettings,
	planningAsOf time.Time,
) *apierror.APIError {
	settingsSnapshot, err := json.Marshal(effective.Settings)
	if err != nil {
		return apierror.NewInternalError(err, "Could not snapshot schedule settings.")
	}
	diagnostics, err := json.Marshal(output.Diagnostics)
	if err != nil {
		return apierror.NewInternalError(err, "Could not snapshot schedule diagnostics.")
	}

	horizonStart := scheduleWeekStart(planningAsOf, effective.Settings.WeekStartDay)

	return s.repos.NewProductionScheduleRepo().RefreshRegenerated(ctx, &domain.ProductionSchedule{
		ID:               schedule.ID,
		AccountID:        accountID,
		PlanningAsOf:     planningAsOf,
		HorizonStartDate: horizonStart,
		HorizonEndDate:   horizonStart.AddDate(0, 0, effective.Settings.HorizonWeeks*7-1),
		HorizonWeeks:     safeconv.IntToInt32(effective.Settings.HorizonWeeks),
		FrozenWeeks:      safeconv.IntToInt32(effective.Settings.FrozenWeeks),
		DemandBasisCode:  effective.DemandBasisCode,
		SolverVersion:    output.SolverVersion,
		SettingsSnapshot: settingsSnapshot,
		Diagnostics:      diagnostics,
	})
}
