package service

import (
	"context"
	"sort"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/tracing"
)

// baselineFor picks the published version that was live for a given week.
//
// Rows arrive newest-publish-first, so the first version whose horizon covers the week is the candidate. For a week that has already ended it must also have been published on or before that week began: skipping that test would let a mid-horizon republish silently rewrite a week the floor had already worked, and the number would change without anyone touching the past.
//
// A week still in progress is not history yet, so the test does not apply to it. The plan being worked right now IS the current published version, whenever during the week it was published — requiring it to predate Monday made every schedule published mid-week report nothing planned, which reads as a broken page rather than as a rule being enforced.
func baselineFor(baselines []domain.AttainmentBaselineRow, week time.Time, now time.Time) *domain.AttainmentBaselineRow {
	weekHasEnded := !week.AddDate(0, 0, 7).After(now)

	for i := range baselines {
		b := &baselines[i]
		if b.PublishedAt == nil {
			continue
		}
		if weekHasEnded && b.PublishedAt.After(week) {
			continue
		}
		if b.HorizonStartDate.After(week) || b.HorizonEndDate.Before(week) {
			continue
		}
		return b
	}
	return nil
}

// ratio returns nil when the denominator is zero. A week nobody planned has no attainment; reporting 0% would read as a total miss.
func ratio(numerator, denominator float64) *float64 {
	if denominator <= 0 {
		return nil
	}
	value := (numerator / denominator) * 100
	return &value
}

type attainmentAccumulator struct {
	planned   float64
	actual    float64
	matched   float64
	waste     float64
	unplanned float64
	runHours  float64
	lines     int64
	batches   int64
	week      *time.Time
}

// buildScheduleAttainment measures actual production against the plan that was live at the time, never against whatever is published now: each week is attributed to the baseline chosen by baselineFor, so a republish cannot rewrite last month's performance.
func (s *analyticsSvcImpl) buildScheduleAttainment(ctx context.Context, params domain.AnalyzeScheduleAttainmentParams) (*domain.ScheduleAttainmentResult, *apierror.APIError) {
	ctx, span := analyticsSvcTracer.Start(ctx, "service.analytics.build_schedule_attainment")
	defer span.End()

	repo := s.repos.NewScheduleAttainmentRepo()

	// Weeks bucket on the account's configured week start, the same day schedule horizons are built on. A fixed Monday would split a plant whose week starts midweek across two buckets, judging each plan week against a fraction of its own output.
	settings, apiErr := s.repos.NewProductionScheduleRepo().GetSettings(ctx, params.AccountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	weekStartDay := int(settings.WeekStartDay)

	windowStart := scheduleWeekStart(params.StartDate, weekStartDay)
	windowEnd := params.EndDate
	// Read once so every week in the window is judged against the same instant.
	now := time.Now().UTC()

	baselines, apiErr := repo.SelectAttainmentBaselines(ctx, domain.SelectAttainmentBaselinesParams{
		AccountID:   params.AccountID,
		WindowStart: windowStart,
		WindowEnd:   windowEnd,
	})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	result := &domain.ScheduleAttainmentResult{
		StartDate: params.StartDate,
		EndDate:   params.EndDate,
		GroupBy:   params.GroupBy,
		// Empty rather than nil: a caller mapping over these should get an empty list, not a null.
		BaselineScheduleIDs: []string{},
		Buckets:             []domain.AttainmentBucket{},
		FrozenAdherence:     []domain.FrozenAdherence{},
		HasBaseline:         len(baselines) > 0,
	}
	if len(baselines) == 0 {
		// Nothing was ever published over this window, so there is no plan to measure against. Returning zeros would look like total failure rather than no data.
		return result, nil
	}

	machineFilter := toSet(params.MachineIDs)
	departmentFilter := toSet(params.DepartmentIDs)

	// planned is keyed by the tuple attainment matches on: (week, machine, item).
	type plannedKey struct {
		week    time.Time
		machine string
		item    string
	}
	planned := map[plannedKey]*attainmentAccumulator{}
	departmentByKey := map[plannedKey]string{}
	usedBaselines := map[string]*domain.AttainmentBaselineRow{}
	// Every machine the plan asked for anywhere in the window. Actuals are read through this set, so the score covers the machines that were scheduled and nothing else: a plant that schedules two knitting machines and scans a hundred other work centres was otherwise measuring the whole factory against a plan that only ever covered two of it.
	//
	// Window-wide rather than per week on purpose. A scheduled machine that was given no work in some week is idle by plan, so what it ran that week is unplanned output — which is the signal. Scoping per week would hide that production entirely instead.
	scheduledMachines := map[string]bool{}

	for i := range baselines {
		b := &baselines[i]

		rows, apiErr := repo.SumPlannedByWeek(ctx, domain.SumPlannedByWeekParams{
			AccountID:            params.AccountID,
			ProductionScheduleID: b.ScheduleID,
			WindowStart:          windowStart,
			WindowEnd:            windowEnd,
		})
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}

		for _, row := range rows {
			week := scheduleWeekStart(row.WeekStartDate, weekStartDay)
			// Each week is attributed to exactly one baseline. A version that covers the week but was not the live plan for it contributes nothing.
			chosen := baselineFor(baselines, week, now)
			if chosen == nil || chosen.ScheduleID != b.ScheduleID {
				continue
			}
			if !passesFilter(machineFilter, row.MachineID) {
				continue
			}
			if row.DepartmentID != nil && !passesFilter(departmentFilter, *row.DepartmentID) {
				continue
			}

			usedBaselines[b.ScheduleID] = b
			scheduledMachines[row.MachineID] = true

			key := plannedKey{week: week, machine: row.MachineID, item: row.ItemID}
			acc := planned[key]
			if acc == nil {
				acc = &attainmentAccumulator{week: &week}
				planned[key] = acc
			}
			acc.planned += row.PlannedQuantity
			acc.runHours += row.PlannedRunHours
			acc.lines += row.LineCount
			if row.DepartmentID != nil {
				departmentByKey[key] = *row.DepartmentID
			}
		}
	}

	actuals, apiErr := repo.SumActualsByWeek(ctx, domain.SumActualsByWeekParams{
		AccountID:    params.AccountID,
		WindowStart:  windowStart,
		WindowEnd:    windowEnd,
		WeekStartDay: weekStartDay,
	})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	for _, row := range actuals {
		week := scheduleWeekStart(row.WeekStartDate, weekStartDay)
		machineID := ""
		if row.MachineID != nil {
			machineID = *row.MachineID
		}
		if !passesFilter(machineFilter, machineID) {
			continue
		}
		if !scheduledMachines[machineID] {
			continue
		}
		departmentID := ""
		if row.DepartmentID != nil {
			departmentID = *row.DepartmentID
		}
		if departmentID != "" && !passesFilter(departmentFilter, departmentID) {
			continue
		}

		key := plannedKey{week: week, machine: machineID, item: row.ItemID}
		acc := planned[key]
		if acc == nil {
			// Production with no matching planned line. This is the schedule-breaker number, so it is accumulated and surfaced rather than dropped.
			acc = &attainmentAccumulator{week: &week}
			planned[key] = acc
			if departmentID != "" {
				departmentByKey[key] = departmentID
			}
		}

		actual := row.ActualQuantity
		acc.actual += actual
		acc.waste += row.WasteQuantity
		acc.batches += row.BatchCount
		if acc.planned <= 0 {
			acc.unplanned += actual
		}
	}

	// Work the floor ran inside a frozen window that the frozen plan never asked for.
	//
	// Counted per (week, machine, item) tuple rather than per scan, so a campaign someone ran across four doffs is one breach of the commitment rather than four. A frozen week is a promise about what the scheduled machines would run; scanning something else onto them breaks it exactly as a hand edit does, so both land in the same score.
	offPlanLines := map[string]int64{}
	offPlanUnits := map[string]float64{}
	for _, acc := range planned {
		if acc.planned > 0 || acc.actual <= 0 || acc.week == nil {
			continue
		}
		baseline := baselineFor(baselines, *acc.week, now)
		if baseline == nil || baseline.FrozenThroughDate == nil || acc.week.After(*baseline.FrozenThroughDate) {
			continue
		}
		offPlanLines[baseline.ScheduleID]++
		offPlanUnits[baseline.ScheduleID] += acc.actual
	}

	// Fold the match-level tuples up into whichever dimension the caller asked for.
	grouped := map[string]*attainmentAccumulator{}
	labels := map[string]string{}

	for key, acc := range planned {
		// LEAST(actual, planned) per tuple, not per bucket: capping after aggregation would let an over-built SKU offset an under-built one and the ratio would stop meaning adherence.
		acc.matched = acc.actual
		if acc.planned < acc.matched {
			acc.matched = acc.planned
		}

		var bucketKey string
		switch params.GroupBy {
		case string(constants.AttainmentGroupByMachine):
			bucketKey = key.machine
		case string(constants.AttainmentGroupByDepartment):
			bucketKey = departmentByKey[key]
		case string(constants.AttainmentGroupByItem):
			bucketKey = key.item
		default:
			bucketKey = key.week.Format(time.RFC3339)
		}

		target := grouped[bucketKey]
		if target == nil {
			target = &attainmentAccumulator{}
			if params.GroupBy == string(constants.AttainmentGroupByWeek) {
				w := key.week
				target.week = &w
			}
			grouped[bucketKey] = target
			labels[bucketKey] = bucketKey
		}
		target.planned += acc.planned
		target.actual += acc.actual
		target.matched += acc.matched
		target.waste += acc.waste
		target.unplanned += acc.unplanned
		target.runHours += acc.runHours
		target.lines += acc.lines
		target.batches += acc.batches
	}

	// Keys are type IDs (or week timestamps); resolve them to the names the UI should show.
	if apiErr := resolveBucketLabels(ctx, repo, params.AccountID, params.GroupBy, labels); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	keys := make([]string, 0, len(grouped))
	for k := range grouped {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	totals := attainmentAccumulator{}
	for _, k := range keys {
		acc := grouped[k]
		totals.planned += acc.planned
		totals.actual += acc.actual
		totals.matched += acc.matched
		totals.waste += acc.waste
		totals.unplanned += acc.unplanned
		totals.runHours += acc.runHours
		totals.lines += acc.lines
		totals.batches += acc.batches

		result.Buckets = append(result.Buckets, toBucket(k, labels[k], acc))
	}
	result.Totals = toBucket("total", "Total", &totals)
	result.ScheduledMachineCount = int64(len(scheduledMachines))

	for id := range usedBaselines {
		result.BaselineScheduleIDs = append(result.BaselineScheduleIDs, id)
	}
	sort.Strings(result.BaselineScheduleIDs)

	// Frozen adherence covers every published version that overlapped the window, not just the ones attainment could attribute weeks to. A version published mid-week is not the baseline for that week, but its frozen commitment still existed and still either held or did not.
	allBaselines := map[string]*domain.AttainmentBaselineRow{}
	allIDs := make([]string, 0, len(baselines))
	for i := range baselines {
		b := &baselines[i]
		if _, seen := allBaselines[b.ScheduleID]; seen {
			continue
		}
		allBaselines[b.ScheduleID] = b
		allIDs = append(allIDs, b.ScheduleID)
	}
	sort.Strings(allIDs)

	if len(allIDs) > 0 {
		adherence, apiErr := frozenAdherence(ctx, repo, params.AccountID, allIDs, allBaselines, offPlanLines, offPlanUnits)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		result.FrozenAdherence = adherence
	}

	return result, nil
}

func toBucket(key, label string, acc *attainmentAccumulator) domain.AttainmentBucket {
	return domain.AttainmentBucket{
		Key:               key,
		Label:             label,
		WeekStartDate:     acc.week,
		PlannedQuantity:   acc.planned,
		ActualQuantity:    acc.actual,
		MatchedQuantity:   acc.matched,
		WasteQuantity:     acc.waste,
		UnplannedQuantity: acc.unplanned,
		PlannedRunHours:   acc.runHours,
		PlannedLines:      acc.lines,
		BatchCount:        acc.batches,
		AttainmentPct:     ratio(acc.matched, acc.planned),
		OutputRatioPct:    ratio(acc.actual, acc.planned),
	}
}

// resolveBucketLabels replaces type-id keys with human-readable names for the active group-by. Unresolved IDs keep the key as a fallback so a missing join never blanks the chart; empty keys (production with no machine/department) become "Unassigned".
func resolveBucketLabels(
	ctx context.Context,
	repo domain.ScheduleAttainmentRepo,
	accountID string,
	groupBy string,
	labels map[string]string,
) *apierror.APIError {
	if len(labels) == 0 {
		return nil
	}

	if _, ok := labels[""]; ok {
		labels[""] = "Unassigned"
	}

	ids := make([]string, 0, len(labels))
	for id := range labels {
		if id == "" {
			continue
		}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return nil
	}

	switch groupBy {
	case string(constants.AttainmentGroupByMachine):
		rows, apiErr := repo.GetMachineLabels(ctx, accountID, ids)
		if apiErr != nil {
			return apiErr
		}
		for _, row := range rows {
			labels[row.ID] = row.Label
		}

	case string(constants.AttainmentGroupByDepartment):
		rows, apiErr := repo.GetDepartmentLabels(ctx, accountID, ids)
		if apiErr != nil {
			return apiErr
		}
		for _, row := range rows {
			labels[row.ID] = row.Label
		}

	case string(constants.AttainmentGroupByItem):
		rows, apiErr := repo.GetItemLabels(ctx, accountID, ids)
		if apiErr != nil {
			return apiErr
		}
		for _, row := range rows {
			labels[row.ID] = row.Label
		}
	}

	return nil
}

// frozenAdherence scores how much of each published commitment survived its frozen week, counting both the hand edits recorded as deviations and the work the floor ran that the frozen plan never called for.
func frozenAdherence(
	ctx context.Context,
	repo domain.ScheduleAttainmentRepo,
	accountID string,
	scheduleIDs []string,
	baselines map[string]*domain.AttainmentBaselineRow,
	offPlanLines map[string]int64,
	offPlanUnits map[string]float64,
) ([]domain.FrozenAdherence, *apierror.APIError) {
	rows, apiErr := repo.CountDeviationsForBaselines(ctx, accountID, scheduleIDs)
	if apiErr != nil {
		return nil, apiErr
	}

	byID := map[string]domain.AttainmentDeviationRow{}
	for _, row := range rows {
		byID[row.ProductionScheduleID] = row
	}

	out := make([]domain.FrozenAdherence, 0, len(scheduleIDs))
	for _, id := range scheduleIDs {
		baseline := baselines[id]
		if baseline == nil {
			continue
		}

		counts := byID[id]
		added := counts.AddedCount
		deviated := float64(counts.DeviationCount) - added
		frozenLines := float64(baseline.FrozenLineCount)
		frozenUnits := baseline.FrozenPlannedQuantity

		offPlan := float64(offPlanLines[id])

		entry := domain.FrozenAdherence{
			ScheduleID:            id,
			Version:               baseline.Version,
			FrozenLineCount:       int64(baseline.FrozenLineCount),
			FrozenPlannedQuantity: frozenUnits,
			DeviatedLines:         int64(deviated),
			AddedLines:            int64(added),
			AbsDeltaUnits:         counts.AbsDeltaQuantity,
			OffPlanLines:          offPlanLines[id],
			OffPlanQuantity:       offPlanUnits[id],
		}
		if baseline.FrozenThroughDate != nil {
			entry.FrozenThroughAt = baseline.FrozenThroughDate
		}

		// Added and off-plan lines sit in BOTH the numerator and the denominator: neither was ever committed to, so each counts against adherence, but each also enlarges what the week turned out to contain, so they cannot push the ratio below zero.
		if denominator := frozenLines + added + offPlan; denominator > 0 {
			value := (1 - (deviated+added+offPlan)/denominator) * 100
			entry.LineAdherence = &value
		}
		if frozenUnits > 0 {
			value := (1 - (entry.AbsDeltaUnits+entry.OffPlanQuantity)/frozenUnits) * 100
			entry.UnitsAdherence = &value
		}

		out = append(out, entry)
	}

	return out, nil
}

func toSet(values []string) map[string]bool {
	if len(values) == 0 {
		return nil
	}
	set := make(map[string]bool, len(values))
	for _, v := range values {
		set[v] = true
	}
	return set
}

// passesFilter treats an empty filter as "no filter", not "match nothing".
func passesFilter(filter map[string]bool, value string) bool {
	if filter == nil {
		return true
	}
	return filter[value]
}
