//go:build e2e

package api_test

import (
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

	for _, key := range []string{"periods", "backlog"} {
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

	after := analyzeDelivery(t, deliveryWindow())
	afterOverall, _ := after["overall"].(map[string]any)
	afterCommitted, _ := afterOverall["committed_order_count"].(float64)

	assert.Greater(t, afterCommitted, beforeCommitted,
		"an issued order with a commitment must enter the measurement")

	// It has not shipped, so it counts against on-time rather than being held back.
	notShipped, _ := afterOverall["not_yet_shipped_count"].(float64)
	assert.Positive(t, notShipped, "an unshipped committed order must be visible as unshipped")
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
