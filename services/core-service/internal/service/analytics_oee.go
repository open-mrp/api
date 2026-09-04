package service

import (
	"context"
	"sort"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/tracing"
)

// buildOeeByDepartment composes the per-department OEE result from the raw reads: unit counts, estimated runtime, logged downtime and the settings-derived planned time.
func (s *analyticsSvcImpl) buildOeeByDepartment(ctx context.Context, params domain.AnalyzeOeeParams) ([]domain.OeeDepartment, *apierror.APIError) {
	ctx, span := analyticsSvcTracer.Start(ctx, "service.analytics.build_oee_by_department")
	defer span.End()

	repo := s.repos.NewAnalyticsRepo()
	window := domain.GetOeeWindowParams{
		AccountID: params.AccountID,
		StartDate: params.StartDate,
		EndDate:   params.EndDate,
	}

	// Scheduled time is the machine-hours the plant actually put on the production schedule, unless the caller supplied its own. A department with no published plan over the window has no availability rather than a denominator guessed from a shift pattern. The machines that carried those hours scope Performance and Quality to the same plant Availability is measured against.
	plannedHours := params.PlannedTimeHours
	var scheduledMachines map[string]bool
	if len(plannedHours) == 0 {
		// Weeks bucket on the account's configured week start, the same day the schedule stores its lines against, so the hours land in the week the window actually covers.
		settings, apiErr := s.repos.NewProductionScheduleRepo().GetSettings(ctx, params.AccountID)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		byWeek, apiErr := s.scheduledHoursByWeek(ctx, params.AccountID, params.StartDate, params.EndDate, int(settings.WeekStartDay))
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		plannedHours = proratedScheduledHours(byWeek, params.StartDate, params.EndDate)

		scheduledMachines, apiErr = s.scheduledMachineIDs(ctx, params.AccountID, params.StartDate, params.EndDate, int(settings.WeekStartDay))
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	rows, apiErr := repo.GetOeeDepartmentData(ctx, window)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// A scheduled department is measured on its scheduled machines alone: output scanned on machines the plan never listed would divide by a run time those machines never contributed to, reporting a department running many times its own speed. Both its output and its Operating Time (run time) are read scoped to those machines and swapped in per scheduled department below; unscheduled departments keep the whole-floor reads.
	scopedByDept := map[string]domain.OeeDepartmentDataRow{}
	scopedRuntimeByDept := map[string]float64{}
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

		scopedRuntimeRows, apiErr := repo.GetOeeEstimatedRuntimeForMachines(ctx, scopedWindow)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		for _, row := range scopedRuntimeRows {
			scopedRuntimeByDept[row.DepartmentID] = row.RuntimeSeconds
		}
	}

	runtimeRows, apiErr := repo.GetOeeEstimatedRuntime(ctx, window)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	downtimeRows, apiErr := repo.GetOeeDowntimeByDepartment(ctx, window)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Build runtime map: departmentID -> runtimeSeconds.
	runtimeMap := make(map[string]float64, len(runtimeRows))
	for _, row := range runtimeRows {
		runtimeMap[row.DepartmentID] = row.RuntimeSeconds
	}

	downtimeMap := aggregateOeeDowntime(downtimeRows)

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

		// A scheduled department reports only its scheduled machines' output and run time; every other department keeps the whole-floor reads. A scheduled department whose machines produced nothing scopes to zero — no Performance rather than a borrowed one.
		data := row
		runtimeSeconds := runtimeMap[row.DepartmentID]
		if len(scheduledMachines) > 0 && plannedHours[row.DepartmentID] > 0 {
			scoped := scopedByDept[row.DepartmentID]
			data.GoodUnits = scoped.GoodUnits
			data.WasteUnits = scoped.WasteUnits
			data.SecondsUnits = scoped.SecondsUnits
			data.StandardSecondsEarned = scoped.StandardSecondsEarned
			runtimeSeconds = scopedRuntimeByDept[row.DepartmentID]
		}

		dept := domain.OeeDepartment{
			DepartmentID:          row.DepartmentID,
			DepartmentName:        row.DepartmentName,
			GoodUnits:             data.GoodUnits,
			WasteUnits:            data.WasteUnits,
			SecondsUnits:          data.SecondsUnits,
			StandardSecondsEarned: data.StandardSecondsEarned,
			EstimatedRuntimeHours: runtimeSeconds / 3600,
		}
		applyOeeDowntime(&dept, downtimeMap[row.DepartmentID])
		computeOeeRatios(&dept, plannedHours[row.DepartmentID], runtimeSeconds)
		departments = append(departments, dept)
	}

	if departments == nil {
		departments = []domain.OeeDepartment{}
	}

	return departments, nil
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

// aggregateOeeDowntime rolls the per-reason downtime rows up per department. Reasons are kept alongside the bucket totals so a Pareto can be rendered without a second query.
func aggregateOeeDowntime(rows []domain.OeeDowntimeRow) map[string]*oeeDowntimeTotals {
	out := make(map[string]*oeeDowntimeTotals)
	for _, row := range rows {
		totals, ok := out[row.DepartmentID]
		if !ok {
			totals = &oeeDowntimeTotals{}
			out[row.DepartmentID] = totals
		}

		// A clipped interval can only be negative if the overlap guard in the query failed; treat that as zero rather than letting it subtract from a loss total.
		seconds := float64(row.DowntimeSeconds)
		if seconds < 0 {
			seconds = 0
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

		totals.events += row.EventCount
		totals.reasons = append(totals.reasons, domain.OeeDowntimeReason{
			ReasonCode:      row.ReasonCode,
			OeeBucket:       row.OeeBucket,
			DowntimeSeconds: seconds,
			EventCount:      row.EventCount,
		})
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

// computeOeeRatios derives Availability x Performance x Quality (Hopp & Spearman, Factory Physics) as a chain of nested time ratios over one clock — the scheduled machines' measured run time:
//
//	scheduled  = planned - not_scheduled           (time nobody planned to run is removed,
//	                                                not counted as a loss)
//	operating  = measured run time of the scheduled machines   (operatingSeconds)
//	run_time   = min(operating, scheduled)         (operating counted toward availability;
//	                                                time run past the schedule is overrun)
//	A = run_time / scheduled                        (<= 1)
//	P = standard_seconds_earned / operating         (<= 1: ideal time for the output cannot
//	                                                exceed the time it was actually running)
//	Q = good / (good + waste + seconds)
//
// Operating Time is measured, not inferred from planned time minus logged downtime. That is what made Performance exceed 100%: the numerator counted every unit the scheduled machines actually scanned while the denominator was capped at planned hours, so a plant that out-ran its schedule read as running faster than its own design. Measuring run time from the same machines' scans puts numerator and denominator on one clock, so P is a true speed ratio bounded by physics.
//
// Availability caps run time at scheduled: run time beyond the schedule is overrun (OverrunSeconds), a schedule-adherence signal reported apart, not extra availability — so A and therefore OEE stay <= 100%. Performance divides by the full measured operating time, so overtime shows as the honest speed it was, not as availability. Because operating spans first-to-last scan, minor stops inside the run stay in Performance's denominator and surface as speed loss, which is where they belong.
//
// Every ratio is left nil when its denominator is zero: an unscheduled department has no OEE, which is not the same as 0% OEE. Quality needs no planned time, so it is computed whenever its own inputs exist.
func computeOeeRatios(dept *domain.OeeDepartment, plannedHours, operatingSeconds float64) {
	// Seconds-grade units count as output but not as good: they are sellable, and they are not first-pass quality. Leaving them out of the denominator would report a plant producing nothing but irregulars as 100% quality.
	totalUnits := dept.GoodUnits + dept.WasteUnits + dept.SecondsUnits
	if totalUnits > 0 {
		quality := dept.GoodUnits / totalUnits
		dept.QualityPct = &quality
	}

	if plannedHours > 0 {
		scheduled := plannedHours*3600 - dept.NotScheduledSeconds
		if scheduled > 0 {
			dept.ScheduledSeconds = scheduled

			operating := operatingSeconds
			if operating < 0 {
				operating = 0
			}
			dept.OperatingTimeSeconds = operating

			// Availability counts operating time only up to the schedule; the rest is overrun, reported apart so overtime never reads as more than 100% available.
			runTime := operating
			if runTime > scheduled {
				dept.OverrunSeconds = operating - scheduled
				runTime = scheduled
			}
			dept.RunTimeSeconds = runTime

			availability := runTime / scheduled
			dept.AvailabilityPct = &availability

			// Performance divides by the full measured run time, not the capped one, so a machine that ran overtime is judged on how fast it ran the whole time it was running.
			if operating > 0 && dept.StandardSecondsEarned > 0 {
				performance := dept.StandardSecondsEarned / operating
				dept.PerformancePct = &performance
				// P > 1 means the scheduled machines earned more standard time than they were measured running — impossible at a correct rate, so it flags a stale or optimistic labor rate. Report it rather than clamp, so the data-quality problem stays visible.
				dept.HasPerformanceAnomaly = performance > 1
			}
		}
	}

	if dept.AvailabilityPct != nil && dept.PerformancePct != nil && dept.QualityPct != nil {
		oee := *dept.AvailabilityPct * *dept.PerformancePct * *dept.QualityPct
		dept.OeePct = &oee
	}
}

// scheduledHoursByWeek reads the account's actual production schedule and returns the Planned Production Time per department for each week in the window.
//
// The number OEE availability divides by is the machine-hours the plant actually scheduled, not a shift pattern multiplied out over the range. This is what a shift-pattern formula got wrong three ways at once: it counted every calendar week in the range whether or not anything was scheduled, counted every machine row whether or not it still runs, and assumed the account's shift settings were current. Reading the plan removes all three — a week nobody planned contributes nothing, and only machines that received work carry hours.
//
// Each week is attributed to the baseline that was live for it — the same choice schedule attainment makes — so a mid-window republish cannot restate a past week's availability. Weeks are kept apart, keyed on the account's own week start, so the per-department table can prorate them (proratedScheduledHours) and the trend can read them one at a time.
func (s *analyticsSvcImpl) scheduledHoursByWeek(ctx context.Context, accountID string, start, end time.Time, weekStartDay int) (map[time.Time]map[string]float64, *apierror.APIError) {
	ctx, span := analyticsSvcTracer.Start(ctx, "service.analytics.scheduled_hours_by_week")
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
		return nil, tracing.Trace(span, apiErr)
	}

	byWeek := map[time.Time]map[string]float64{}
	for i := range baselines {
		b := &baselines[i]

		rows, apiErr := repo.SumScheduledHoursByDepartmentWeek(ctx, domain.SumPlannedByWeekParams{
			AccountID:            accountID,
			ProductionScheduleID: b.ScheduleID,
			WindowStart:          windowStart,
			WindowEnd:            windowEnd,
		})
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}

		for _, row := range rows {
			week := scheduleWeekStart(row.WeekStartDate, weekStartDay)
			// One baseline owns each week; a version that covers the week but was not the live plan for it contributes nothing, exactly as in attainment.
			chosen := baselineFor(baselines, week, now)
			if chosen == nil || chosen.ScheduleID != b.ScheduleID {
				continue
			}

			deptHours := byWeek[week]
			if deptHours == nil {
				deptHours = map[string]float64{}
				byWeek[week] = deptHours
			}
			// Changeover is scheduled machine time too, so it is part of the denominator; the changeover that then runs is charged back as an availability loss. Leaving it out here while subtracting logged changeover downtime would double-count it and understate availability.
			deptHours[row.DepartmentID] += row.PlannedRunHours + row.PlannedChangeoverMinutes/60
		}
	}
	return byWeek, nil
}

// weekOverlapFraction is how much of one production week [weekMonday, weekMonday+7) falls inside [start, end], as a fraction in [0, 1].
//
// The plan is week-granular — it says how many hours a week held, not which day held them — so a window shorter than a week takes a proportional slice of that week's hours. This is the same slice a shift-pattern formula took by measuring days against seven; it just applies it to the hours the plant actually scheduled rather than to a theoretical weekly capacity. A window that covers a whole week takes all of it.
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

// proratedScheduledHours folds the per-week schedule down to one Planned Production Time per department for the whole window, taking each week's hours in proportion to how much of that week the window covers. It is the denominator the per-department table reports the window against.
func proratedScheduledHours(byWeek map[time.Time]map[string]float64, start, end time.Time) map[string]float64 {
	out := map[string]float64{}
	for week, deptHours := range byWeek {
		fraction := weekOverlapFraction(week, start, end)
		if fraction <= 0 {
			continue
		}
		for departmentID, hours := range deptHours {
			out[departmentID] += hours * fraction
		}
	}
	return out
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

// scheduledMachineIDs returns every machine the account's live plan put on the schedule across the window — the machines OEE measures. Performance divides the standard time output earned by the scheduled machines' run time, so output from machines the plan never listed has to be kept out of the numerator or a department reads as running many times its own speed.
//
// Each week is attributed to the same baseline scheduledHoursByWeek and attainment choose, so all three agree on which plan owned a week and therefore on which machines it scheduled.
func (s *analyticsSvcImpl) scheduledMachineIDs(ctx context.Context, accountID string, start, end time.Time, weekStartDay int) (map[string]bool, *apierror.APIError) {
	ctx, span := analyticsSvcTracer.Start(ctx, "service.analytics.scheduled_machine_ids")
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
		return nil, tracing.Trace(span, apiErr)
	}

	machines := map[string]bool{}
	for i := range baselines {
		b := &baselines[i]

		rows, apiErr := repo.SumPlannedByWeek(ctx, domain.SumPlannedByWeekParams{
			AccountID:            accountID,
			ProductionScheduleID: b.ScheduleID,
			WindowStart:          windowStart,
			WindowEnd:            windowEnd,
		})
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}

		for _, row := range rows {
			week := scheduleWeekStart(row.WeekStartDate, weekStartDay)
			// One baseline owns each week, exactly as in scheduledHoursByWeek: a version that covered the week but was not its plan scheduled nothing for it.
			chosen := baselineFor(baselines, week, now)
			if chosen == nil || chosen.ScheduleID != b.ScheduleID {
				continue
			}
			if row.MachineID != "" {
				machines[row.MachineID] = true
			}
		}
	}
	return machines, nil
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
