package service

import (
	"context"
	"sort"
	"time"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/tracing"
)

// buildOeeByDepartment composes the per-department OEE result from the raw reads: unit counts, estimated runtime, logged downtime, batch-ticket scan intervals and the settings-derived planned time.
func (s *analyticsSvcImpl) buildOeeByDepartment(ctx context.Context, params domain.AnalyzeOeeParams) ([]domain.OeeDepartment, *apierror.APIError) {
	ctx, span := analyticsSvcTracer.Start(ctx, "service.analytics.build_oee_by_department")
	defer span.End()

	repo := s.repos.NewAnalyticsRepo()
	window := domain.GetOeeWindowParams{
		AccountID: params.AccountID,
		StartDate: params.StartDate,
		EndDate:   params.EndDate,
	}

	rows, apiErr := repo.GetOeeDepartmentData(ctx, window)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	runtimeRows, apiErr := repo.GetOeeEstimatedRuntime(ctx, window)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	downtimeRows, apiErr := repo.GetOeeDowntimeByDepartment(ctx, window)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	scanRows, apiErr := repo.GetOeeScanIntervals(ctx, window)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	downtimeIntervalRows, apiErr := repo.GetOeeMachineDowntimeIntervals(ctx, window)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	measuredByDepartment := computeMeasuredPerformance(scanRows, buildMachineDowntime(downtimeIntervalRows))

	// Build runtime map: departmentID -> runtimeSeconds.
	runtimeMap := make(map[string]float64, len(runtimeRows))
	for _, row := range runtimeRows {
		runtimeMap[row.DepartmentID] = row.RuntimeSeconds
	}

	downtimeMap := aggregateOeeDowntime(downtimeRows)

	// Scheduled time is derived from the account's shift pattern unless the caller supplied its own. Nobody should have to hand-enter a denominator the settings already state.
	plannedHours := params.PlannedTimeHours
	if len(plannedHours) == 0 {
		settings, apiErr := s.repos.NewProductionScheduleRepo().GetSettings(ctx, params.AccountID)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}

		machineRows, apiErr := repo.CountMachinesByDepartment(ctx, params.AccountID)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		machinesByDepartment := make(map[string]int64, len(machineRows))
		for _, row := range machineRows {
			machinesByDepartment[row.DepartmentID] = row.MachineCount
		}

		plannedHours = derivePlannedHours(settings, machinesByDepartment, params.StartDate, params.EndDate)
	}

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

		runtimeSeconds := runtimeMap[row.DepartmentID]

		dept := domain.OeeDepartment{
			DepartmentID:          row.DepartmentID,
			DepartmentName:        row.DepartmentName,
			GoodUnits:             row.GoodUnits,
			WasteUnits:            row.WasteUnits,
			SecondsUnits:          row.SecondsUnits,
			StandardSecondsEarned: row.StandardSecondsEarned,
			EstimatedRuntimeHours: runtimeSeconds / 3600,
		}
		applyOeeDowntime(&dept, downtimeMap[row.DepartmentID])
		applyMeasuredPerformance(&dept, measuredByDepartment[row.DepartmentID])
		computeOeeRatios(&dept, plannedHours[row.DepartmentID])
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

// A scan gap only counts as continuous production when it is no longer than oeeGapOutlierMultiplier x the ticket's own ideal time, with a floor of oeeGapMinimumCapSeconds so quick tickets are not excluded by ordinary idle minutes. Longer gaps are breaks in production (overnight, weekend, between campaigns), not slow running, and charging them to Performance would make P meaningless.
const (
	oeeGapOutlierMultiplier = 4.0
	oeeGapMinimumCapSeconds = 2 * 3600.0
)

// oeeDowntimeInterval is one machine's logged downtime window, used to reduce the scan gap it overlaps.
type oeeDowntimeInterval struct {
	start time.Time
	end   time.Time
}

// oeeMeasuredSample accumulates one department's Performance sample: the ideal time of the tickets whose scan gap qualified, and the actual time those gaps took.
type oeeMeasuredSample struct {
	idealSeconds  float64
	actualSeconds float64
	tickets       int64
}

// buildMachineDowntime indexes logged downtime by machine, keeping only the buckets that Performance must not be charged for: availability, quality and not-scheduled losses are already accounted elsewhere in the OEE arithmetic, so leaving them inside a scan gap would double-count them. Performance-bucket downtime (minor stops, reduced speed) stays in the gap — that loss is exactly what measured Performance exists to capture. Overlapping intervals are merged so stacked events cannot subtract the same seconds twice.
func buildMachineDowntime(rows []domain.OeeMachineDowntimeIntervalRow) map[string][]oeeDowntimeInterval {
	out := make(map[string][]oeeDowntimeInterval)
	for _, row := range rows {
		if row.OeeBucket == domain.OeeBucketPerformance || row.EndedAt == nil {
			continue
		}
		if !row.EndedAt.After(row.StartedAt) {
			continue
		}
		out[row.MachineID] = append(out[row.MachineID], oeeDowntimeInterval{
			start: row.StartedAt,
			end:   *row.EndedAt,
		})
	}

	for machineID, intervals := range out {
		sort.Slice(intervals, func(i, j int) bool { return intervals[i].start.Before(intervals[j].start) })
		merged := intervals[:0]
		for _, iv := range intervals {
			if len(merged) > 0 && !iv.start.After(merged[len(merged)-1].end) {
				if iv.end.After(merged[len(merged)-1].end) {
					merged[len(merged)-1].end = iv.end
				}
				continue
			}
			merged = append(merged, iv)
		}
		out[machineID] = merged
	}

	return out
}

// downtimeWithinGap is the number of logged-downtime seconds inside (gapStart, gapEnd) for one machine, clipped to the gap.
func downtimeWithinGap(intervals []oeeDowntimeInterval, gapStart, gapEnd time.Time) float64 {
	var total float64
	for _, iv := range intervals {
		start := iv.start
		if start.Before(gapStart) {
			start = gapStart
		}
		end := iv.end
		if end.After(gapEnd) {
			end = gapEnd
		}
		if end.After(start) {
			total += end.Sub(start).Seconds()
		}
	}
	return total
}

// computeMeasuredPerformance turns batch-ticket scans into a measured Performance sample per department.
//
// The time a ticket actually took is the gap between its scan and the previous scan off the same machine: a ticket scanned at 7:10 whose predecessor came off at 6:00 took 70 minutes, and if its output should have taken 60 minutes at the step's ideal cycle time, that ticket ran at 60/70. Logged downtime inside the gap is subtracted first so a loss already charged to Availability is not charged to Performance too.
//
// A ticket is dropped from the sample — but still serves as the next ticket's predecessor timestamp — when it has no ideal time configured, no predecessor in the window, or its gap fails the outlier cap above. The first ticket per machine in any window is therefore never sampled; over a week-long window that loss is one gap per machine.
func computeMeasuredPerformance(
	rows []domain.OeeScanIntervalRow,
	downtimeByMachine map[string][]oeeDowntimeInterval,
) map[string]*oeeMeasuredSample {
	out := make(map[string]*oeeMeasuredSample)

	var prevMachine string
	var prevScan time.Time
	var havePrev bool

	for _, row := range rows {
		if row.ScannedAt == nil {
			continue
		}
		if row.MachineID != prevMachine {
			prevMachine = row.MachineID
			havePrev = false
		}

		scan := *row.ScannedAt
		if havePrev {
			ideal := row.IdealSeconds
			rawGap := scan.Sub(prevScan).Seconds()
			if ideal > 0 && rawGap > 0 {
				actual := rawGap - downtimeWithinGap(downtimeByMachine[row.MachineID], prevScan, scan)
				gapCap := oeeGapOutlierMultiplier * ideal
				if gapCap < oeeGapMinimumCapSeconds {
					gapCap = oeeGapMinimumCapSeconds
				}
				if actual > 0 && actual <= gapCap {
					sample, ok := out[row.DepartmentID]
					if !ok {
						sample = &oeeMeasuredSample{}
						out[row.DepartmentID] = sample
					}
					sample.idealSeconds += ideal
					sample.actualSeconds += actual
					sample.tickets++
				}
			}
		}

		prevScan = scan
		havePrev = true
	}

	return out
}

// applyMeasuredPerformance copies a department's scan-interval sample onto its result row.
func applyMeasuredPerformance(dept *domain.OeeDepartment, sample *oeeMeasuredSample) {
	if sample == nil {
		return
	}
	dept.MeasuredIdealSeconds = sample.idealSeconds
	dept.MeasuredRunSeconds = sample.actualSeconds
	dept.PerformanceTicketCount = sample.tickets
}

// computeOeeRatios derives Availability x Performance x Quality from planned time, measured downtime and the scan-interval Performance sample.
//
//	scheduled = planned - not_scheduled      (time nobody planned to run is removed,
//	                                          not counted as a loss)
//	run_time  = scheduled - availability_loss
//	A = run_time / scheduled
//	P = ideal_seconds / actual_seconds       (measured across batch-ticket scan gaps;
//	                                          falls back to earned / run_time when no
//	                                          gap qualified in the window)
//	Q = good / (good + waste)
//
// Every ratio is left nil when its denominator is zero: an unscheduled department has no OEE, which is not the same as 0% OEE. Quality and measured Performance need no planned time, so they are computed whenever their own inputs exist.
func computeOeeRatios(dept *domain.OeeDepartment, plannedHours float64) {
	// Seconds-grade units count as output but not as good: they are sellable, and they are not first-pass quality. Leaving them out of the denominator would report a plant producing nothing but irregulars as 100% quality.
	totalUnits := dept.GoodUnits + dept.WasteUnits + dept.SecondsUnits
	if totalUnits > 0 {
		quality := dept.GoodUnits / totalUnits
		dept.QualityPct = &quality
	}

	// Performance is measured: the time the sampled tickets should have taken over the time their scan gaps say they took. Independent of planned time entirely.
	if dept.MeasuredRunSeconds > 0 && dept.MeasuredIdealSeconds > 0 {
		performance := dept.MeasuredIdealSeconds / dept.MeasuredRunSeconds
		dept.PerformancePct = &performance
		dept.PerformanceBasis = string(constants.OeePerformanceBasisScanIntervals)
		// P > 1 means tickets came off faster than the ideal cycle time allows, which is either a stale ideal or a missed scan collapsing two tickets into one gap. Report it and flag it; clamping would hide the data-quality problem.
		dept.HasPerformanceAnomaly = performance > 1
	}

	if plannedHours > 0 {
		scheduled := plannedHours*3600 - dept.NotScheduledSeconds
		if scheduled > 0 {
			dept.ScheduledSeconds = scheduled

			runTime := scheduled - dept.AvailabilityLossSeconds
			if runTime < 0 {
				runTime = 0
			}
			dept.RunTimeSeconds = runTime

			availability := runTime / scheduled
			dept.AvailabilityPct = &availability

			// Fallback Performance for windows where no scan gap qualified (single scans, unlinked machines): standard time earned over estimated run time, the old shift-pattern estimate. Kept so a department without machine links still reports something, and labelled so the caller can tell the difference.
			if dept.PerformancePct == nil && runTime > 0 && dept.StandardSecondsEarned > 0 {
				performance := dept.StandardSecondsEarned / runTime
				dept.PerformancePct = &performance
				dept.PerformanceBasis = string(constants.OeePerformanceBasisRunTimeEstimate)
				dept.HasPerformanceAnomaly = performance > 1
			}
		}
	}

	if dept.AvailabilityPct != nil && dept.PerformancePct != nil && dept.QualityPct != nil {
		oee := *dept.AvailabilityPct * *dept.PerformancePct * *dept.QualityPct
		dept.OeePct = &oee
	}
}

// derivePlannedHours works out each department's scheduled time from the account's own shift pattern, so nobody has to type it into a browser.
//
// Scheduled time is machine-hours, not wall-clock hours: downtime is logged per machine, so a three-machine room measured against one machine's shift would report availability three times worse than it is.
//
// The shift pattern comes from the production-schedule settings — the same assumption the solver sizes capacity with — so OEE availability and schedule utilisation are answering the same question rather than quietly disagreeing.
func derivePlannedHours(
	settings *domain.ProductionScheduleSettings,
	machinesByDepartment map[string]int64,
	start, end time.Time,
) map[string]float64 {
	out := map[string]float64{}
	if settings == nil {
		return out
	}

	hoursPerWeek := float64(settings.ShiftsPerDay) * settings.HoursPerShift * float64(settings.WorkDaysPerWeek)
	if hoursPerWeek <= 0 {
		return out
	}

	// A three-day window is measured against three days of shift, not a whole week.
	days := end.Sub(start).Hours() / 24
	if days <= 0 {
		return out
	}
	weeks := days / 7

	for departmentID, machineCount := range machinesByDepartment {
		if machineCount <= 0 {
			continue
		}
		out[departmentID] = hoursPerWeek * weeks * float64(machineCount)
	}
	return out
}
