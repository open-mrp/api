package service

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/tracing"
)

// buildOeeTrend measures the same OEE arithmetic as buildOeeByDepartment, one production week at a time, so a plant can see whether a number is moving rather than only what it is today.
//
// Everything is read once for the whole window and bucketed in memory: a week-per-round-trip loop would multiply four queries by the length of the range, and a year of weekly points would be 200 queries for one chart.
//
// Only departments with scheduled time take part. A department with no machines — including the 'unassigned' bucket that batches with no scanning-station department fall into — has no Availability and therefore no OEE, exactly as in the per-department table; counting its output in Quality while it cannot appear in Availability or Performance would make the three terms describe different plants.
func (s *analyticsSvcImpl) buildOeeTrend(ctx context.Context, params domain.AnalyzeOeeTrendParams) ([]domain.OeeTrendPeriod, *apierror.APIError) {
	ctx, span := analyticsSvcTracer.Start(ctx, "service.analytics.build_oee_trend")
	defer span.End()

	repo := s.repos.NewAnalyticsRepo()

	// The week scans bucket into follows the account's configured week start, so a point on this chart covers the same days as the schedule week beside it, and the downtime and planned-hours arithmetic below charge the same weeks. Read first because the bucketed output query needs it; it is a single-row fetch, so serializing it ahead of the heavy reads costs nothing measurable.
	settings, apiErr := s.repos.NewProductionScheduleRepo().GetSettings(ctx, params.AccountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	weekStartDay := int(settings.WeekStartDay)

	window := domain.GetOeeWindowParams{
		AccountID:    params.AccountID,
		StartDate:    params.StartDate,
		EndDate:      params.EndDate,
		WeekStartDay: weekStartDay,
	}

	// These reads share no inputs, and the scan aggregate over the window dominates the others combined — running them in sequence spends the whole chart's latency budget waiting on one query while the rest idle. Errors are collected and the first non-nil is returned, so failure behaves exactly as it did when these ran in order.
	var (
		outputRows   []domain.OeeTrendDepartmentWeekRow
		downtimeRows []domain.OeeDowntimeIntervalRow
		machineRows  []domain.DepartmentMachineCountRow
		errs         [3]*apierror.APIError
		wg           sync.WaitGroup
	)

	wg.Add(3)
	go func() {
		defer wg.Done()
		outputRows, errs[0] = repo.GetOeeTrendDepartmentDataByWeek(ctx, window)
	}()
	go func() {
		defer wg.Done()
		downtimeRows, errs[1] = repo.GetOeeTrendDowntimeIntervals(ctx, window)
	}()
	go func() {
		defer wg.Done()
		machineRows, errs[2] = repo.CountMachinesByDepartment(ctx, params.AccountID)
	}()
	wg.Wait()

	for _, apiErr := range errs {
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	deptFilter := make(map[string]bool, len(params.DepartmentIDs))
	for _, id := range params.DepartmentIDs {
		deptFilter[id] = true
	}

	machinesByDepartment := make(map[string]int64, len(machineRows))
	for _, row := range machineRows {
		if len(deptFilter) > 0 && !deptFilter[row.DepartmentID] {
			continue
		}
		machinesByDepartment[row.DepartmentID] = row.MachineCount
	}

	outputByWeek := indexOeeTrendOutput(outputRows, deptFilter, weekStartDay)

	periods := []domain.OeeTrendPeriod{}
	for _, bucket := range oeeTrendBuckets(params.StartDate, params.EndDate, weekStartDay) {
		plannedHours := derivePlannedHours(settings, machinesByDepartment, bucket.start, bucket.end)
		downtime := oeeTrendDowntimeInBucket(downtimeRows, deptFilter, bucket.start, bucket.end)
		periods = append(periods, buildOeeTrendPeriod(bucket, plannedHours, outputByWeek[scheduleWeekStart(bucket.start, weekStartDay)], downtime))
	}

	return periods, nil
}

// oeeTrendBucket is one production week, clipped to the requested window.
type oeeTrendBucket struct {
	start time.Time
	end   time.Time
}

// oeeTrendBuckets cuts a window into production weeks, clipped at both ends.
//
// Weeks start on the account's configured week start, the same key SumActualsByWeek buckets scans into, so a point on this chart covers the same days as the schedule week it sits next to. The first and last weeks of a range are usually partial — a range ending today ends mid-week — and are reported over the part that was asked for rather than padded out to a full week, because planned time scales with the days in the bucket and a padded week would report a plant that had stopped.
func oeeTrendBuckets(start, end time.Time, weekStartDay int) []oeeTrendBucket {
	if !end.After(start) {
		return nil
	}

	var buckets []oeeTrendBucket
	for cursor := start; cursor.Before(end); {
		next := scheduleWeekStart(cursor, weekStartDay).AddDate(0, 0, 7)
		if next.After(end) {
			next = end
		}
		buckets = append(buckets, oeeTrendBucket{start: cursor, end: next})
		cursor = next
	}
	return buckets
}

// indexOeeTrendOutput groups the per-week output rows by their week key, dropping departments the caller filtered out.
func indexOeeTrendOutput(rows []domain.OeeTrendDepartmentWeekRow, deptFilter map[string]bool, weekStartDay int) map[time.Time]map[string]domain.OeeTrendDepartmentWeekRow {
	out := make(map[time.Time]map[string]domain.OeeTrendDepartmentWeekRow)
	for _, row := range rows {
		if len(deptFilter) > 0 && !deptFilter[row.DepartmentID] {
			continue
		}
		week := scheduleWeekStart(row.WeekStart, weekStartDay)
		byDepartment, ok := out[week]
		if !ok {
			byDepartment = map[string]domain.OeeTrendDepartmentWeekRow{}
			out[week] = byDepartment
		}
		byDepartment[row.DepartmentID] = row
	}
	return out
}

// oeeTrendDowntimeTotals is one department's downtime inside one week, split by the OEE term each reason charges.
type oeeTrendDowntimeTotals struct {
	availability float64
	notScheduled float64
	events       int64
}

// oeeTrendDowntimeInBucket clips every logged interval to one week and totals it per department.
//
// An event that spans midnight on Sunday belongs partly to each week, so it is clipped rather than assigned whole to the week it started in — otherwise a Friday breakdown running into Monday would make one week look worse and the next look untouched.
func oeeTrendDowntimeInBucket(rows []domain.OeeDowntimeIntervalRow, deptFilter map[string]bool, bucketStart, bucketEnd time.Time) map[string]*oeeTrendDowntimeTotals {
	out := make(map[string]*oeeTrendDowntimeTotals)
	for _, row := range rows {
		if len(deptFilter) > 0 && !deptFilter[row.DepartmentID] {
			continue
		}

		start := row.StartedAt
		if start.Before(bucketStart) {
			start = bucketStart
		}
		end := row.EndedAt
		if end.After(bucketEnd) {
			end = bucketEnd
		}
		if !end.After(start) {
			continue
		}

		totals, ok := out[row.DepartmentID]
		if !ok {
			totals = &oeeTrendDowntimeTotals{}
			out[row.DepartmentID] = totals
		}

		seconds := end.Sub(start).Seconds()
		switch row.OeeBucket {
		case domain.OeeBucketAvailability:
			totals.availability += seconds
		case domain.OeeBucketNotScheduled:
			totals.notScheduled += seconds
		}
		// Performance- and quality-bucket downtime is counted as logged but charged to no denominator: it stays inside run time, where it shows up as speed or quality loss.
		totals.events++
	}
	return out
}

// buildOeeTrendPeriod rolls one week's departments up into a single set of ratios.
//
// Each department's scheduled and run time is derived by computeOeeRatios — the same function the per-department table uses — and only then summed, so the trend and the table can never disagree about what a week was worth. The ratios are recomputed from the summed seconds rather than averaged: averaging department percentages would let a room that ran an hour weigh as heavily as one that ran all week.
func buildOeeTrendPeriod(
	bucket oeeTrendBucket,
	plannedHours map[string]float64,
	output map[string]domain.OeeTrendDepartmentWeekRow,
	downtime map[string]*oeeTrendDowntimeTotals,
) domain.OeeTrendPeriod {
	period := domain.OeeTrendPeriod{StartsAt: bucket.start, EndsAt: bucket.end}

	// Sorted so the roll-up sums in a stable order and two identical requests cannot differ in the last float digit.
	departmentIDs := make([]string, 0, len(plannedHours))
	for departmentID := range plannedHours {
		departmentIDs = append(departmentIDs, departmentID)
	}
	sort.Strings(departmentIDs)

	for _, departmentID := range departmentIDs {
		dept := domain.OeeDepartment{}
		if row, ok := output[departmentID]; ok {
			dept.GoodUnits = row.GoodUnits
			dept.WasteUnits = row.WasteUnits
			dept.SecondsUnits = row.SecondsUnits
			dept.StandardSecondsEarned = row.StandardSecondsEarned
		}
		if totals, ok := downtime[departmentID]; ok {
			dept.AvailabilityLossSeconds = totals.availability
			dept.NotScheduledSeconds = totals.notScheduled
			period.DowntimeEventCount += totals.events
			period.HasDowntimeData = period.HasDowntimeData || totals.events > 0
		}

		computeOeeRatios(&dept, plannedHours[departmentID])
		if dept.ScheduledSeconds <= 0 {
			continue
		}

		period.GoodUnits += dept.GoodUnits
		period.WasteUnits += dept.WasteUnits
		period.SecondsUnits += dept.SecondsUnits
		period.StandardSecondsEarned += dept.StandardSecondsEarned
		period.ScheduledSeconds += dept.ScheduledSeconds
		period.RunTimeSeconds += dept.RunTimeSeconds
		period.AvailabilityLossSeconds += dept.AvailabilityLossSeconds
		period.NotScheduledSeconds += dept.NotScheduledSeconds
	}

	if period.ScheduledSeconds > 0 {
		availability := period.RunTimeSeconds / period.ScheduledSeconds
		period.AvailabilityPct = &availability
	}
	if period.RunTimeSeconds > 0 && period.StandardSecondsEarned > 0 {
		performance := period.StandardSecondsEarned / period.RunTimeSeconds
		period.PerformancePct = &performance
	}
	if totalUnits := period.GoodUnits + period.WasteUnits + period.SecondsUnits; totalUnits > 0 {
		quality := period.GoodUnits / totalUnits
		period.QualityPct = &quality
	}
	if period.AvailabilityPct != nil && period.PerformancePct != nil && period.QualityPct != nil {
		oee := *period.AvailabilityPct * *period.PerformancePct * *period.QualityPct
		period.OeePct = &oee
	}

	return period
}
