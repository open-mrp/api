package service

import (
	"context"
	"sort"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/tracing"
)

// buildOeeByDepartment composes the per-department OEE result from the raw reads: unit counts, logged downtime and the shift-configuration capacity the scheduled machines carried.
func (s *analyticsSvcImpl) buildOeeByDepartment(ctx context.Context, params domain.AnalyzeOeeParams) ([]domain.OeeDepartment, *apierror.APIError) {
	ctx, span := analyticsSvcTracer.Start(ctx, "service.analytics.build_oee_by_department")
	defer span.End()

	repo := s.repos.NewAnalyticsRepo()
	window := domain.GetOeeWindowParams{
		AccountID: params.AccountID,
		StartDate: params.StartDate,
		EndDate:   params.EndDate,
	}

	// Read unconditionally: the shift window comes off the same settings row and is needed to clip downtime even when the caller supplied its own capacity.
	settings, apiErr := s.repos.NewProductionScheduleRepo().GetSettings(ctx, params.AccountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	shiftWindow := newOeeShiftWindow(settings)

	// Capacity is the machine-hours the scheduled machines could run over the window — their shift configuration, not a scan span — unless the caller supplied its own. A department with no published plan has no scheduled machines and so no capacity, which is no OEE rather than a guessed denominator. The scheduled machines scope Performance and Quality to the same plant Availability is measured against.
	capacityHours := params.PlannedTimeHours
	var scheduledMachines map[string]bool
	if len(capacityHours) == 0 {
		perMachineWeekly := machineWeeklyCapacityHours(settings)
		capacityByWeek, machines, apiErr := s.scheduledCapacity(ctx, params.AccountID, params.StartDate, params.EndDate, int(settings.WeekStartDay), perMachineWeekly)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		scheduledMachines = machines

		// Fold the per-week capacity down to one window figure per department, prorating a partial
		// first or last week. Capacity is summed per week rather than as (machines unioned over the
		// window) x window length, so a machine scheduled for only part of the range carries only the
		// weeks it was actually planned for — the same figure the trend reports for the same window.
		capacityHours = map[string]float64{}
		for week, deptHours := range capacityByWeek {
			fraction := weekOverlapFraction(week, params.StartDate, params.EndDate)
			if fraction <= 0 {
				continue
			}
			for departmentID, hours := range deptHours {
				capacityHours[departmentID] += hours * fraction
			}
		}
	}

	rows, apiErr := repo.GetOeeDepartmentData(ctx, window)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// A scheduled department is measured on its scheduled machines alone: output scanned on machines the plan never listed would divide by a capacity those machines never contributed, reporting a department running many times its own speed. Its output is read scoped to those machines and swapped in per scheduled department below; unscheduled departments keep the whole-floor reads.
	scopedByDept := map[string]domain.OeeDepartmentDataRow{}
	if len(scheduledMachines) > 0 {
		scopedWindow := window
		scopedWindow.MachineIDs = machineSlice(scheduledMachines)
		scopedRows, apiErr := repo.GetOeeDepartmentData(ctx, scopedWindow)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		for _, row := range scopedRows {
			scopedByDept[row.DepartmentID] = row
		}
	}

	downtimeRows, apiErr := repo.GetOeeDowntimeIntervals(ctx, window)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	downtimeMap := aggregateOeeDowntime(downtimeRows, params.StartDate, params.EndDate, shiftWindow)

	// Build department filter set for optional filtering.
	deptFilter := make(map[string]bool, len(params.DepartmentIDs))
	for _, id := range params.DepartmentIDs {
		deptFilter[id] = true
	}

	var departments []domain.OeeDepartment
	for _, row := range rows {
		if len(deptFilter) > 0 && !deptFilter[row.DepartmentID] {
			continue
		}

		// A scheduled department reports only its scheduled machines' output; every other department keeps the whole-floor reads. A scheduled department whose machines produced nothing scopes to zero — no Performance rather than a borrowed one.
		data := row
		if len(scheduledMachines) > 0 && capacityHours[row.DepartmentID] > 0 {
			scoped := scopedByDept[row.DepartmentID]
			data.GoodUnits = scoped.GoodUnits
			data.WasteUnits = scoped.WasteUnits
			data.SecondsUnits = scoped.SecondsUnits
			data.StandardSecondsEarned = scoped.StandardSecondsEarned
		}

		dept := domain.OeeDepartment{
			DepartmentID:          row.DepartmentID,
			DepartmentName:        row.DepartmentName,
			GoodUnits:             data.GoodUnits,
			WasteUnits:            data.WasteUnits,
			SecondsUnits:          data.SecondsUnits,
			StandardSecondsEarned: data.StandardSecondsEarned,
		}
		applyOeeDowntime(&dept, downtimeMap[row.DepartmentID])
		computeOeeRatios(&dept, capacityHours[row.DepartmentID])
		departments = append(departments, dept)
	}

	if departments == nil {
		departments = []domain.OeeDepartment{}
	}

	return departments, nil
}

// machineWeeklyCapacityHours is the hours one machine can run in a week from the account's shift configuration: shifts per day x hours per shift x work days per week. The changeover headroom the scheduler reserves is deliberately left out — OEE measures against the time the equipment could actually run, and the changeover that is then worked is charged back as an availability loss, not removed from capacity up front.
func machineWeeklyCapacityHours(settings *domain.ProductionScheduleSettings) float64 {
	if settings == nil {
		return 0
	}
	return float64(settings.ShiftsPerDay) * settings.HoursPerShift * float64(settings.WorkDaysPerWeek)
}

// oeeDowntimeTotals is one department's downtime, already split by the OEE term each reason charges.
type oeeDowntimeTotals struct {
	availability float64
	performance  float64
	quality      float64
	notScheduled float64
	changeover   float64
	events       int64
	reasons      []domain.OeeDowntimeReason
}

// aggregateOeeDowntime rolls logged downtime intervals up per department, charging each only the part that falls inside the reporting window AND inside the plant's shift window. Reasons are kept alongside the bucket totals so a Pareto can be rendered without a second query.
//
// Both clips matter and neither is optional. The reporting clip is obvious. The shift clip is what keeps run time honest: capacity counts only the hours the plant is open, so a stop has to be measured against the same hours or it removes time the denominator never held. A 16-hour breakdown logged 15:00 Thursday to 07:00 Friday against a 06:00-22:00 plant is 7 hours of lost Thursday and 1 of Friday, not 16. A nil window (an account that has never configured one) charges the whole span, which is the behaviour before the window existed.
func aggregateOeeDowntime(rows []domain.OeeDowntimeIntervalRow, windowStart, windowEnd time.Time, shift *oeeShiftWindow) map[string]*oeeDowntimeTotals {
	out := make(map[string]*oeeDowntimeTotals)
	for _, row := range rows {
		start := row.StartedAt
		if start.Before(windowStart) {
			start = windowStart
		}
		end := row.EndedAt
		if end.After(windowEnd) {
			end = windowEnd
		}

		seconds := shift.OverlapSeconds(start, end)
		// An event wholly outside the shift is not a loss and must not be counted as an occurrence either, or the Pareto fills with stops that cost nothing.
		if seconds <= 0 {
			continue
		}

		totals, ok := out[row.DepartmentID]
		if !ok {
			totals = &oeeDowntimeTotals{}
			out[row.DepartmentID] = totals
		}

		switch row.OeeBucket {
		case domain.OeeBucketAvailability:
			totals.availability += seconds
		case domain.OeeBucketPerformance:
			totals.performance += seconds
		case domain.OeeBucketQuality:
			totals.quality += seconds
		case domain.OeeBucketNotScheduled:
			totals.notScheduled += seconds
		}

		if row.ReasonCode == domain.MachineDowntimeReasonCodeChangeover {
			totals.changeover += seconds
		}

		// One interval is one event. The reasons list is per-event here and folded below, because two events sharing a reason can clip to different seconds.
		totals.events++
		totals.reasons = append(totals.reasons, domain.OeeDowntimeReason{
			ReasonCode:      row.ReasonCode,
			OeeBucket:       row.OeeBucket,
			DowntimeSeconds: seconds,
			EventCount:      1,
		})
	}

	// Fold the per-event rows down to one row per reason, the shape the Pareto is rendered from and the shape the SQL aggregate used to return.
	for departmentID, totals := range out {
		byReason := make(map[string]*domain.OeeDowntimeReason, len(totals.reasons))
		order := make([]string, 0, len(totals.reasons))
		for _, reason := range totals.reasons {
			existing, ok := byReason[reason.ReasonCode]
			if !ok {
				copied := reason
				byReason[reason.ReasonCode] = &copied
				order = append(order, reason.ReasonCode)
				continue
			}
			existing.DowntimeSeconds += reason.DowntimeSeconds
			existing.EventCount++
		}
		folded := make([]domain.OeeDowntimeReason, 0, len(order))
		for _, code := range order {
			folded = append(folded, *byReason[code])
		}
		out[departmentID].reasons = folded
	}

	// Largest loss first, then by code so the order is stable across identical totals.
	for _, totals := range out {
		sort.Slice(totals.reasons, func(i, j int) bool {
			if totals.reasons[i].DowntimeSeconds != totals.reasons[j].DowntimeSeconds {
				return totals.reasons[i].DowntimeSeconds > totals.reasons[j].DowntimeSeconds
			}
			return totals.reasons[i].ReasonCode < totals.reasons[j].ReasonCode
		})
	}

	return out
}

// applyOeeDowntime copies a department's measured downtime onto its result row.
func applyOeeDowntime(dept *domain.OeeDepartment, totals *oeeDowntimeTotals) {
	if totals == nil {
		dept.DowntimeBreakdown = []domain.OeeDowntimeReason{}
		return
	}

	dept.AvailabilityLossSeconds = totals.availability
	dept.PerformanceLossSeconds = totals.performance
	dept.QualityLossSeconds = totals.quality
	dept.NotScheduledSeconds = totals.notScheduled
	dept.ChangeoverSeconds = totals.changeover
	dept.DowntimeEventCount = totals.events
	dept.DowntimeBreakdown = totals.reasons
	dept.HasDowntimeData = totals.events > 0
}

// computeOeeRatios derives Availability x Performance x Quality (Hopp & Spearman, Factory Physics) as a chain of nested time ratios over one clock — the scheduled machines' capacity net of downtime:
//
//	capacity   = scheduled machines' shift-configuration hours over the window (capacityHours)
//	ppt        = capacity - not_scheduled          (time nobody planned to run is removed,
//	                                                not counted as a loss)
//	run_time   = ppt - availability downtime        (breakdowns and changeover both land in the
//	                                                availability bucket, so both come out here)
//	A = run_time / ppt                              (<= 1)
//	P = standard_seconds_earned / run_time          (<= 1: ideal time for the output cannot
//	                                                exceed the time the equipment was running)
//	Q = good / (good + waste + seconds)
//
// Run time is inferred from capacity minus logged downtime, not measured from the first-to-last scan span. The scan span is what made Performance exceed 100%: batch or end-of-shift scanning collapses the span to minutes while the earned standard time is hours, and a single-scan day spans zero seconds, so the denominator ran systematically small and P floated above 1. Deriving run time from the shift configuration puts numerator and denominator on one clock — capacity — so P is a true speed ratio bounded by physics.
//
// Because run time is capacity minus downtime it can never exceed the planned production time, so A and therefore OEE stay <= 100% without an overrun cap; OverrunSeconds is retired and left zero. Small stops that were never logged as downtime stay inside run time, where they surface as Performance (speed) loss, which is where they belong.
//
// Every ratio is left nil when its denominator is zero: an unscheduled department has no capacity and so no OEE, which is not the same as 0% OEE. Quality needs no capacity, so it is computed whenever its own inputs exist.
func computeOeeRatios(dept *domain.OeeDepartment, capacityHours float64) {
	// Seconds-grade units count as output but not as good: they are sellable, and they are not first-pass quality. Leaving them out of the denominator would report a plant producing nothing but irregulars as 100% quality.
	totalUnits := dept.GoodUnits + dept.WasteUnits + dept.SecondsUnits
	if totalUnits > 0 {
		quality := dept.GoodUnits / totalUnits
		dept.QualityPct = &quality
	}

	if capacityHours > 0 {
		// Planned Production Time is the scheduled machines' capacity net of the time nobody planned to run.
		ppt := capacityHours*3600 - dept.NotScheduledSeconds
		if ppt < 0 {
			ppt = 0
		}
		if ppt > 0 {
			dept.ScheduledSeconds = ppt

			// Run time is planned time the equipment was actually running: everything left after the availability stops (breakdowns and changeover) are taken out.
			runTime := ppt - dept.AvailabilityLossSeconds
			if runTime < 0 {
				runTime = 0
			}
			dept.RunTimeSeconds = runTime
			// Operating and run time are one clock now — capacity minus downtime — so Performance divides by the same time Availability does.
			dept.OperatingTimeSeconds = runTime
			dept.EstimatedRuntimeHours = runTime / 3600

			availability := runTime / ppt
			dept.AvailabilityPct = &availability

			if runTime > 0 && dept.StandardSecondsEarned > 0 {
				performance := dept.StandardSecondsEarned / runTime
				dept.PerformancePct = &performance
				// P > 1 means the scheduled machines earned more standard time than they were running — impossible at a correct rate, so it flags a stale or optimistic labor rate, under-logged downtime, or a mis-set capacity. Report it rather than clamp, so the data-quality problem stays visible.
				dept.HasPerformanceAnomaly = performance > 1
			}
		}
	}

	if dept.AvailabilityPct != nil && dept.PerformancePct != nil && dept.QualityPct != nil {
		oee := *dept.AvailabilityPct * *dept.PerformancePct * *dept.QualityPct
		dept.OeePct = &oee
	}
}

// weekOverlapFraction is how much of one production week [weekMonday, weekMonday+7) falls inside [start, end], as a fraction in [0, 1].
//
// Capacity is week-granular — it is how many machine-hours a week held, not which day held them — so a window shorter than a week takes a proportional slice of that week's capacity. A window that covers a whole week takes all of it.
func weekOverlapFraction(weekMonday, start, end time.Time) float64 {
	weekEnd := weekMonday.AddDate(0, 0, 7)
	from := weekMonday
	if start.After(from) {
		from = start
	}
	to := weekEnd
	if end.Before(to) {
		to = end
	}
	if !to.After(from) {
		return 0
	}
	return to.Sub(from).Hours() / (7 * 24)
}

// scaleDeptHours multiplies every department's hours by a factor, used to prorate one week's schedule to a partial-week trend bucket.
func scaleDeptHours(hours map[string]float64, factor float64) map[string]float64 {
	if factor == 1 {
		return hours
	}
	out := make(map[string]float64, len(hours))
	for departmentID, h := range hours {
		out[departmentID] = h * factor
	}
	return out
}

// filterDeptHours drops departments the caller did not ask for. An empty filter means no filter, matching passesFilter.
func filterDeptHours(hours map[string]float64, deptFilter map[string]bool) map[string]float64 {
	if len(deptFilter) == 0 {
		return hours
	}
	out := make(map[string]float64, len(hours))
	for departmentID, h := range hours {
		if deptFilter[departmentID] {
			out[departmentID] = h
		}
	}
	return out
}

// scheduledCapacity reads the account's live plan once and returns two things over the window:
// the Planned Production Time capacity per department for each production week, and the flat set of
// every machine the plan scheduled anywhere in the window.
//
// Capacity is kept per week — a department's distinct scheduled machines that week times one
// machine's shift-configuration hours — rather than unioned across the window and multiplied out.
// A machine planned for only part of the range must carry only the weeks it was scheduled for, or
// Planned Production Time is inflated and the per-department table disagrees with the trend on the
// same window. The per-department table folds these weeks down with weekOverlapFraction; the trend
// reads them one at a time. A machine listed on several lines in a week counts its capacity once.
//
// The flat machine set scopes the output reads: Performance divides the standard time earned by the
// scheduled machines' run time, so output from machines the plan never listed has to be kept out of
// the numerator. It is unioned across the window on purpose — a machine's output is measured wherever
// it scanned, even in a week it was idle by plan.
//
// Each week is attributed to the baseline that governed it, the same choice attainment makes, so all
// three agree on which plan owned a week. A machine scheduled under no department is dropped from
// capacity: an unassigned department has no availability.
func (s *analyticsSvcImpl) scheduledCapacity(ctx context.Context, accountID string, start, end time.Time, weekStartDay int, perMachineWeekly float64) (map[time.Time]map[string]float64, map[string]bool, *apierror.APIError) {
	ctx, span := analyticsSvcTracer.Start(ctx, "service.analytics.scheduled_capacity")
	defer span.End()

	repo := s.repos.NewScheduleAttainmentRepo()
	windowStart := scheduleWeekStart(start, weekStartDay)
	windowEnd := end
	// Read once so every week is judged against the same instant.
	now := time.Now().UTC()

	baselines, apiErr := repo.SelectAttainmentBaselines(ctx, domain.SelectAttainmentBaselinesParams{
		AccountID:   accountID,
		WindowStart: windowStart,
		WindowEnd:   windowEnd,
	})
	if apiErr != nil {
		return nil, nil, tracing.Trace(span, apiErr)
	}

	// Distinct scheduled machines per (week, department), plus the flat union for output scoping.
	machinesByWeekDept := map[time.Time]map[string]map[string]bool{}
	flat := map[string]bool{}
	rows, apiErr := repo.SumPlannedByWeek(ctx, domain.SumPlannedByWeekParams{
		AccountID:             accountID,
		ProductionScheduleIDs: baselineScheduleIDs(baselines),
		WindowStart:           windowStart,
		WindowEnd:             windowEnd,
	})
	if apiErr != nil {
		return nil, nil, tracing.Trace(span, apiErr)
	}

	for _, row := range rows {
		week := scheduleWeekStart(row.WeekStartDate, weekStartDay)
		// One baseline owns each week: a version that covered the week but was not its live plan scheduled nothing for it.
		chosen := baselineFor(baselines, week, now)
		if chosen == nil || chosen.ScheduleID != row.ProductionScheduleID {
			continue
		}
		if row.MachineID == "" || row.DepartmentID == nil || *row.DepartmentID == "" {
			continue
		}
		flat[row.MachineID] = true
		byDept := machinesByWeekDept[week]
		if byDept == nil {
			byDept = map[string]map[string]bool{}
			machinesByWeekDept[week] = byDept
		}
		set := byDept[*row.DepartmentID]
		if set == nil {
			set = map[string]bool{}
			byDept[*row.DepartmentID] = set
		}
		set[row.MachineID] = true
	}

	capacityByWeek := make(map[time.Time]map[string]float64, len(machinesByWeekDept))
	for week, byDept := range machinesByWeekDept {
		deptHours := make(map[string]float64, len(byDept))
		for departmentID, set := range byDept {
			deptHours[departmentID] = float64(len(set)) * perMachineWeekly
		}
		capacityByWeek[week] = deptHours
	}
	return capacityByWeek, flat, nil
}

// machineSlice is the sorted machine-id list an OEE read filters on. Sorted so the query text is stable across identical requests; empty when nothing was scheduled, which the read reads as no machine filter.
func machineSlice(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for id := range set {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}
