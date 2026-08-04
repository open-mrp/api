package service

import (
	"context"
	"sort"
	"time"

	"github.com/augno/api/services/core-service/internal/domain"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/tracing"
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

// computeOeeRatios derives Availability x Performance x Quality from planned time, measured downtime and the ideal cycle times the period's output earned.
//
//	scheduled = planned - not_scheduled      (time nobody planned to run is removed,
//	                                          not counted as a loss)
//	run_time  = scheduled - availability_loss
//	A = run_time / scheduled
//	P = standard_seconds_earned / run_time   (ideal cycle time x units produced, over
//	                                          the time the department was running)
//	Q = good / (good + waste)
//
// Every ratio is left nil when its denominator is zero: an unscheduled department has no OEE, which is not the same as 0% OEE. Quality needs no planned time, so it is computed whenever its own inputs exist.
//
// Performance shares Availability's run time deliberately. Availability answers how long the department was running, Performance how fast it ran while it was: only availability-bucket downtime leaves run time, so minor stops, idling and slow cycles stay inside the denominator and show up as speed loss, which is the only place they belong.
func computeOeeRatios(dept *domain.OeeDepartment, plannedHours float64) {
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

			runTime := scheduled - dept.AvailabilityLossSeconds
			if runTime < 0 {
				runTime = 0
			}
			dept.RunTimeSeconds = runTime

			availability := runTime / scheduled
			dept.AvailabilityPct = &availability

			if runTime > 0 && dept.StandardSecondsEarned > 0 {
				performance := dept.StandardSecondsEarned / runTime
				dept.PerformancePct = &performance
				// P > 1 means the output took less time than its ideal cycle time allows, which is a stale run rate rather than a machine that beat its own design. Report it and flag it; clamping would hide the data-quality problem.
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
