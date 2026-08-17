//go:build e2e

package api_test

import (
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Delivery performance: did we ship what we promised, when we promised it.

const deliveryPerformancePath = "/v1/core/analytics/delivery-performance"

func analyzeDelivery(t *testing.T, body map[string]any) map[string]any {
	t.Helper()

	status, respBody, err := apiClient.Put(deliveryPerformancePath, body)
	require.NoError(t, err)
	require.Less(t, status, 500, "delivery performance must not 5xx: %s", string(respBody))
	requireStatus(t, 200, status, respBody)
	return parseJSON(respBody)
}

func deliveryWindow() map[string]any {
	return map[string]any{
		"starts_at": time.Now().UTC().AddDate(0, 0, -180).Format(time.RFC3339),
		"ends_at":   time.Now().UTC().AddDate(0, 0, 180).Format(time.RFC3339),
	}
}

// The response has to be complete and internally consistent: every count must add
// up, or the tiles built on it will disagree with each other.
func TestDeliveryPerformance_ShapeAndArithmetic(t *testing.T) {
	t.Parallel()

	result := analyzeDelivery(t, deliveryWindow())

	assert.Equal(t, "analyze_delivery_performance_response", jsonField(result, "object"))

	overall, ok := result["overall"].(map[string]any)
	require.True(t, ok, "an overall figure is always returned: %v", result)
	assert.Equal(t, "delivery_performance", jsonField(overall, "object"))

	committed, _ := overall["committed_order_count"].(float64)
	shipped, _ := overall["shipped_order_count"].(float64)
	onTime, _ := overall["on_time_order_count"].(float64)
	onTimeInFull, _ := overall["on_time_in_full_count"].(float64)
	notShipped, _ := overall["not_yet_shipped_count"].(float64)

	assert.LessOrEqual(t, shipped, committed, "more orders cannot ship than were due")
	assert.LessOrEqual(t, onTime, shipped, "an order cannot be on time without shipping")
	assert.LessOrEqual(t, onTimeInFull, onTime, "on-time-in-full is a subset of on-time")
	assert.Equal(t, committed, shipped+notShipped,
		"every due order either shipped or did not")

	for _, key := range []string{"periods", "backlog", "lateness", "by_customer", "by_customer_group", "by_product_line", "by_commitment_source"} {
		list, ok := result[key].(map[string]any)
		require.True(t, ok, "%s must be a list resource, not null: %v", key, result[key])
		_, ok = list["data"].([]any)
		require.True(t, ok, "%s.data must be an array", key)
	}

	_, ok = result["uncommitted_order_count"].(float64)
	assert.True(t, ok, "the count of excluded orders must always be reported")
}

// A window with nothing due must report no rate rather than zero — zero reads as
// total failure, and a quiet period is not a failure.
func TestDeliveryPerformance_NothingDueHasNoRate(t *testing.T) {
	t.Parallel()

	// Far in the past, before any seeded commitment.
	result := analyzeDelivery(t, map[string]any{
		"starts_at": "2019-01-01T00:00:00Z",
		"ends_at":   "2019-02-01T00:00:00Z",
	})

	overall, ok := result["overall"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, float64(0), overall["committed_order_count"])
	assert.Nil(t, overall["on_time_pct"], "no orders due means no on-time rate, not 0%%")
	assert.Nil(t, overall["on_time_in_full_pct"])
	assert.Nil(t, overall["average_days_late"], "no late orders means no average lateness")
}

// An order shipped on time has to move the numbers, and the commitment measured
// against is the one stamped at issue.
func TestDeliveryPerformance_CountsACommittedOrder(t *testing.T) {
	t.Parallel()

	before := analyzeDelivery(t, deliveryWindow())
	beforeCommitted, _ := before["overall"].(map[string]any)["committed_order_count"].(float64)

	customerID := leadTimeCustomer(t, "e2e-delivery", ptrInt(30), "")
	order := issueOrderForCustomer(t, customerID, nil)
	require.NotEmpty(t, shipByDate(t, order), "the order must carry a commitment to be measured")

	// The window covers the whole account, and sibling tests issue and unissue their own committed orders throughout the run, so a single reading taken after ours can land below the baseline through no fault of the order just issued. The order added here stays committed for the rest of the test, so the count only has to clear the baseline once.
	eventually(t, e2eAsyncWaitTimeout, e2eAsyncPollInterval, func() error {
		afterOverall, _ := analyzeDelivery(t, deliveryWindow())["overall"].(map[string]any)
		afterCommitted, _ := afterOverall["committed_order_count"].(float64)
		if afterCommitted <= beforeCommitted {
			return fmt.Errorf("an issued order with a commitment must enter the measurement: committed %v, was %v", afterCommitted, beforeCommitted)
		}
		// It has not shipped, so it counts against on-time rather than being held back.
		if notShipped, _ := afterOverall["not_yet_shipped_count"].(float64); notShipped <= 0 {
			return fmt.Errorf("an unshipped committed order must be visible as unshipped: not_yet_shipped_count %v", notShipped)
		}
		return nil
	})
}

// Buckets are keyed by the date promised, not the date shipped: an order promised in
// March and shipped in May is March's miss.
func TestDeliveryPerformance_BucketsByGranularity(t *testing.T) {
	t.Parallel()

	for _, granularity := range []string{"day", "week", "month"} {
		t.Run(granularity, func(t *testing.T) {
			body := deliveryWindow()
			body["granularity"] = granularity

			result := analyzeDelivery(t, body)
			list, _ := result["periods"].(map[string]any)
			data, _ := list["data"].([]any)

			var previous time.Time
			for _, raw := range data {
				period, ok := raw.(map[string]any)
				require.True(t, ok)
				assert.Equal(t, "delivery_performance", jsonField(period, "object"))

				start := jsonField(period, "period_start")
				require.NotEmpty(t, start, "a bucket must name the period it covers")
				parsed, err := time.Parse(time.RFC3339, start)
				require.NoError(t, err)
				if !previous.IsZero() {
					assert.False(t, parsed.Before(previous), "buckets must come back chronologically")
				}
				previous = parsed
			}
		})
	}
}

// The backlog is work still owed, so its bands are always present even when empty —
// a missing band and an empty one would read differently to a chart.
func TestDeliveryPerformance_BacklogBandsAlwaysPresent(t *testing.T) {
	t.Parallel()

	result := analyzeDelivery(t, deliveryWindow())
	list, _ := result["backlog"].(map[string]any)
	data, _ := list["data"].([]any)

	labels := map[string]bool{}
	for _, raw := range data {
		bucket, ok := raw.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "delivery_backlog_bucket", jsonField(bucket, "object"))
		labels[jsonField(bucket, "label")] = true

		count, _ := bucket["order_count"].(float64)
		units, _ := bucket["units"].(float64)
		assert.GreaterOrEqual(t, count, float64(0))
		assert.GreaterOrEqual(t, units, float64(0))
	}

	for _, want := range []string{"1_7_days", "8_30_days", "31_60_days", "over_60_days"} {
		assert.True(t, labels[want], "the %s band must always be present, even at zero", want)
	}
}

func TestDeliveryPerformance_ValidatesItsWindow(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Put(deliveryPerformancePath, map[string]any{
		"starts_at": time.Now().UTC().Format(time.RFC3339),
		"ends_at":   time.Now().UTC().AddDate(0, 0, -30).Format(time.RFC3339),
	})
	require.NoError(t, err)
	require.Less(t, status, 500, "must not 5xx: %s", string(body))
	assert.Equal(t, 400, status, "a window that ends before it starts should be rejected: %s", string(body))
}

// The buckets and the overall figure are computed from one set of outcomes, so
// they have to agree. A tile showing a total that its own breakdown does not add
// up to is worse than either number alone.
func TestDeliveryPerformance_PeriodsSumToOverall(t *testing.T) {
	t.Parallel()

	result := analyzeDelivery(t, deliveryWindow())
	overall, ok := result["overall"].(map[string]any)
	require.True(t, ok)

	list, _ := result["periods"].(map[string]any)
	data, _ := list["data"].([]any)

	var committed, shipped, onTime, onTimeInFull, notShipped float64
	for _, raw := range data {
		period, ok := raw.(map[string]any)
		require.True(t, ok)

		periodCommitted, _ := period["committed_order_count"].(float64)
		periodShipped, _ := period["shipped_order_count"].(float64)
		periodOnTime, _ := period["on_time_order_count"].(float64)
		periodOnTimeInFull, _ := period["on_time_in_full_count"].(float64)
		periodNotShipped, _ := period["not_yet_shipped_count"].(float64)

		// Each bucket is internally consistent on its own terms too.
		assert.Equal(t, periodCommitted, periodShipped+periodNotShipped,
			"every order due in a period either shipped or did not: %v", period)
		assert.LessOrEqual(t, periodOnTime, periodShipped, "an order cannot be on time without shipping")
		assert.LessOrEqual(t, periodOnTimeInFull, periodOnTime, "on-time-in-full is a subset of on-time")

		// A bucket only exists because something was due in it.
		assert.Positive(t, periodCommitted, "an empty period should not be reported at all: %v", period)

		committed += periodCommitted
		shipped += periodShipped
		onTime += periodOnTime
		onTimeInFull += periodOnTimeInFull
		notShipped += periodNotShipped
	}

	assert.Equal(t, overall["committed_order_count"], committed, "the buckets must account for every due order")
	assert.Equal(t, overall["shipped_order_count"], shipped)
	assert.Equal(t, overall["on_time_order_count"], onTime)
	assert.Equal(t, overall["on_time_in_full_count"], onTimeInFull)
	assert.Equal(t, overall["not_yet_shipped_count"], notShipped)
}

// The rates are the counts, expressed as a share. A rate that drifts from its own
// numerator is the failure this catches: the tile and the drill-down would tell
// different stories about the same orders.
func TestDeliveryPerformance_RatesFollowTheCounts(t *testing.T) {
	t.Parallel()

	// Its own commitment, so the window is never empty and the rates are always
	// actually computed — a test that only checks arithmetic when the rest of the
	// suite happens to have produced some proves nothing when run alone.
	commitOneOrder(t, "e2e-delivery-rates")

	result := analyzeDelivery(t, deliveryWindow())
	overall, ok := result["overall"].(map[string]any)
	require.True(t, ok)

	committed, _ := overall["committed_order_count"].(float64)
	require.Positive(t, committed, "the order issued above must be due inside the window")

	onTime, _ := overall["on_time_order_count"].(float64)
	onTimeInFull, _ := overall["on_time_in_full_count"].(float64)

	onTimePct, ok := overall["on_time_pct"].(float64)
	require.True(t, ok, "orders were due, so a rate must be reported: %v", overall["on_time_pct"])
	assert.InDelta(t, onTime/committed*100, onTimePct, 0.01)

	onTimeInFullPct, ok := overall["on_time_in_full_pct"].(float64)
	require.True(t, ok)
	assert.InDelta(t, onTimeInFull/committed*100, onTimeInFullPct, 0.01)

	// Lateness is averaged over late orders only, so it is only reported when there
	// were some — averaging it over every order would dilute a real problem.
	late, _ := overall["late_order_count"].(float64)
	if late == 0 {
		assert.Nil(t, overall["average_days_late"], "with nothing late there is no lateness to average")
	} else if averageLate, ok := overall["average_days_late"].(float64); ok {
		assert.Positive(t, averageLate, "an order counted late is at least a day late")
	}
}

// Omitting the granularity has to mean something definite. It means weeks, and the
// buckets have to prove it by starting on Mondays — the same day the production
// schedule buckets on, so a delivery week and a plan week name the same seven days.
func TestDeliveryPerformance_DefaultsToWeeklyBuckets(t *testing.T) {
	t.Parallel()

	// The two calls are compared to each other, but the window they measure is live: sibling tests in this package issue and unissue orders throughout the run, and an order that loses its commitment between the calls takes its bucket with it. Retry until one pair of reads sees the same data, which is what the comparison is actually about. A genuine disagreement — a default of days rather than weeks — never converges, because day buckets and week buckets cannot coincide.
	var implicitStarts []string
	eventually(t, e2eAsyncWaitTimeout, e2eAsyncPollInterval, func() error {
		implicit := analyzeDelivery(t, deliveryWindow())

		explicitBody := deliveryWindow()
		explicitBody["granularity"] = "week"
		explicit := analyzeDelivery(t, explicitBody)

		implicitStarts = periodStarts(t, implicit)
		explicitStarts := periodStarts(t, explicit)
		if !slices.Equal(implicitStarts, explicitStarts) {
			return fmt.Errorf("omitting the granularity must mean the same thing as asking for weeks: got %v, want %v", implicitStarts, explicitStarts)
		}
		return nil
	})

	for _, start := range implicitStarts {
		parsed, err := time.Parse(time.RFC3339, start)
		require.NoError(t, err)
		assert.Equal(t, time.Monday, parsed.UTC().Weekday(),
			"a delivery week must start on the same day a plan week does: %s", start)
	}
}

// Day and month buckets have to actually be days and months, or the picker changes
// a label without changing the data behind it.
func TestDeliveryPerformance_GranularityDecidesTheBucketBoundary(t *testing.T) {
	t.Parallel()

	t.Run("month buckets start on the first", func(t *testing.T) {
		body := deliveryWindow()
		body["granularity"] = "month"

		for _, start := range periodStarts(t, analyzeDelivery(t, body)) {
			parsed, err := time.Parse(time.RFC3339, start)
			require.NoError(t, err)
			assert.Equal(t, 1, parsed.UTC().Day(), "a month bucket starts on the first: %s", start)
		}
	})

	t.Run("day buckets are never coarser than week buckets", func(t *testing.T) {
		commitOneOrder(t, "e2e-delivery-buckets")

		dayBody := deliveryWindow()
		dayBody["granularity"] = "day"
		weekBody := deliveryWindow()
		weekBody["granularity"] = "week"

		// The relation holds only between two readings of the same commitments; sibling tests unissue theirs between the calls, which can retire a whole day bucket and leave the day count short of the week count. Retry until both readings see one set of orders.
		eventually(t, e2eAsyncWaitTimeout, e2eAsyncPollInterval, func() error {
			days := len(periodStarts(t, analyzeDelivery(t, dayBody)))
			weeks := len(periodStarts(t, analyzeDelivery(t, weekBody)))
			if weeks == 0 {
				return fmt.Errorf("the order issued above must land in some bucket")
			}
			if days < weeks {
				return fmt.Errorf("the same commitments split by day cannot produce fewer buckets than by week: %d days, %d weeks", days, weeks)
			}
			return nil
		})
	})
}

func TestDeliveryPerformance_RejectsMalformedRequests(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		body map[string]any
	}{
		{"missing start", map[string]any{"ends_at": time.Now().UTC().Format(time.RFC3339)}},
		{"missing end", map[string]any{"starts_at": time.Now().UTC().Format(time.RFC3339)}},
		{"identical bounds", map[string]any{
			"starts_at": "2026-01-01T00:00:00Z",
			"ends_at":   "2026-01-01T00:00:00Z",
		}},
		{"unknown granularity", func() map[string]any {
			body := deliveryWindow()
			body["granularity"] = "fortnight"
			return body
		}()},
		{"unparseable date", map[string]any{"starts_at": "last tuesday", "ends_at": "2026-01-01T00:00:00Z"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			status, body, err := apiClient.Put(deliveryPerformancePath, tc.body)
			require.NoError(t, err)
			require.Less(t, status, 500, "must not 5xx: %s", string(body))
			assert.Equal(t, 400, status, "body: %s", string(body))
		})
	}

	t.Run("unknown field", func(t *testing.T) {
		body := deliveryWindow()
		body[bogusE2EJSONField] = "x"

		status, respBody, err := apiClient.Put(deliveryPerformancePath, body)
		require.NoError(t, err)
		assertJSONUnknownFieldRejected(t, "PUT", deliveryPerformancePath, status, respBody)
	})
}

// One tenant's delivery record must not leak into another's. The account is taken
// from the credential rather than the request, so this is measured: an order
// committed in tenant A must not move a single one of tenant B's counts.
func TestDeliveryPerformance_IsScopedToTheCallingTenant(t *testing.T) {
	t.Parallel()
	clientB := getTenantBClient()

	analyzeAsB := func() map[string]any {
		status, body, err := clientB.Put(deliveryPerformancePath, deliveryWindow())
		require.NoError(t, err)
		require.Less(t, status, 500, "must not 5xx: %s", string(body))
		requireStatus(t, 200, status, body)

		parsed := parseJSON(body)
		assert.Equal(t, "analyze_delivery_performance_response", jsonField(parsed, "object"))
		overall, ok := parsed["overall"].(map[string]any)
		require.True(t, ok, "tenant B gets its own well-formed answer: %s", string(body))
		return overall
	}

	before := analyzeAsB()

	customerID := leadTimeCustomer(t, "e2e-delivery-tenant", ptrInt(30), "")
	order := issueOrderForCustomer(t, customerID, nil)
	require.NotEmpty(t, shipByDate(t, order), "precondition: tenant A committed to something in the window")

	after := analyzeAsB()
	for _, key := range []string{"committed_order_count", "shipped_order_count", "not_yet_shipped_count"} {
		assert.Equal(t, before[key], after[key],
			"a commitment made in tenant A must not appear in tenant B's %s", key)
	}
}

// periodStarts pulls the bucket start dates out of a delivery-performance result.
func periodStarts(t *testing.T, result map[string]any) []string {
	t.Helper()

	list, ok := result["periods"].(map[string]any)
	require.True(t, ok, "periods must be a list resource: %v", result["periods"])
	data, _ := list["data"].([]any)

	out := make([]string, 0, len(data))
	for _, raw := range data {
		period, ok := raw.(map[string]any)
		require.True(t, ok)
		out = append(out, jsonField(period, "period_start"))
	}
	return out
}

// commitOneOrder issues an order carrying a ship-by commitment, so a delivery
// window always has something due to measure.
func commitOneOrder(t *testing.T, prefix string) {
	t.Helper()

	customerID := leadTimeCustomer(t, prefix, ptrInt(30), "")
	order := issueOrderForCustomer(t, customerID, nil)
	require.NotEmpty(t, shipByDate(t, order), "an issued order must carry the commitment being measured")
}

// The lateness bands are the companion to the average: an average cannot tell a day of slippage from a month of it, so the bands must always be present even at zero.
func TestDeliveryPerformance_LatenessBandsAlwaysPresent(t *testing.T) {
	t.Parallel()

	result := analyzeDelivery(t, deliveryWindow())
	list, _ := result["lateness"].(map[string]any)
	data, _ := list["data"].([]any)

	labels := map[string]bool{}
	for _, raw := range data {
		bucket, ok := raw.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "delivery_lateness_bucket", jsonField(bucket, "object"))
		labels[jsonField(bucket, "label")] = true

		count, _ := bucket["order_count"].(float64)
		shipped, _ := bucket["shipped_count"].(float64)
		assert.GreaterOrEqual(t, count, float64(0))
		// A band's shipped orders are a subset of its orders; the remainder are what the backlog still owes.
		assert.LessOrEqual(t, shipped, count, "a band cannot have shipped more orders than it holds")
	}

	for _, want := range []string{"1_3_days", "4_7_days", "8_30_days", "over_30_days"} {
		assert.True(t, labels[want], "the %s band must always be present, even at zero", want)
	}
}

// A breakdown that does not add up to the headline is worse than no breakdown: it makes every drilldown suspect.
func TestDeliveryPerformance_BreakdownsReconcileWithOverall(t *testing.T) {
	t.Parallel()

	result := analyzeDelivery(t, deliveryWindow())
	overall, ok := result["overall"].(map[string]any)
	require.True(t, ok)
	committed, _ := overall["committed_order_count"].(float64)

	// Product lines are deliberately excluded: an order spanning two lines is counted under both, so that dimension sums to more than the total.
	for _, key := range []string{"by_customer", "by_customer_group", "by_commitment_source"} {
		list, ok := result[key].(map[string]any)
		require.True(t, ok, "%s must be a list resource: %v", key, result[key])
		data, _ := list["data"].([]any)

		var sum float64
		var previousLate float64 = -1
		for _, raw := range data {
			row, ok := raw.(map[string]any)
			require.True(t, ok)
			assert.Equal(t, "delivery_breakdown", jsonField(row, "object"))

			performance, ok := row["performance"].(map[string]any)
			require.True(t, ok, "every breakdown row carries its figures: %v", row)
			rowCommitted, _ := performance["committed_order_count"].(float64)
			sum += rowCommitted

			// Worst-first ordering is what makes the table readable without sorting it.
			rowLate, _ := performance["late_order_count"].(float64)
			if previousLate >= 0 {
				assert.LessOrEqual(t, rowLate, previousLate, "%s must be ordered worst-first", key)
			}
			previousLate = rowLate
		}

		assert.Equal(t, committed, sum, "%s must account for every order the overall figure counted", key)
	}
}

// An unknown id is a filter matching nothing, not an error — and it must narrow the excluded count too, or a filtered rate would sit beside an account-wide exclusion.
func TestDeliveryPerformance_FiltersNarrowTheMeasuredSet(t *testing.T) {
	t.Parallel()

	body := deliveryWindow()
	body["customer_ids"] = []string{"acc_definitely_not_a_real_customer"}

	result := analyzeDelivery(t, body)

	overall, ok := result["overall"].(map[string]any)
	require.True(t, ok)
	committed, _ := overall["committed_order_count"].(float64)
	assert.Equal(t, float64(0), committed, "a filter matching no customer measures no orders")
	assert.Nil(t, overall["on_time_pct"], "no orders due means no rate, not 0%")

	uncommitted, _ := result["uncommitted_order_count"].(float64)
	assert.Equal(t, float64(0), uncommitted,
		"the excluded count has to describe the same slice the rates do")
}
