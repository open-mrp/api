//go:build e2e

package api_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Promises on the plan: which campaign is building which order, what the plan does
// not build in time, and the earliest date it could ship something new.

const quotePromiseDatePath = "/v1/operations/production-schedules/actions/quote-promise-date"

// The at-risk report has to describe the version it belongs to, not a fresh solve.
func TestSchedulePromises_AtRiskOrdersDescribeTheVersion(t *testing.T) {
	t.Parallel()

	schedule := generateSchedule(t, map[string]any{})
	scheduleID := jsonField(schedule, "id")
	cleanupSchedule(t, scheduleID)

	status, body, err := apiClient.GetListRaw(productionSchedulesPath+"/"+scheduleID+"/at-risk-orders", nil)
	require.NoError(t, err)
	require.Less(t, status, 500, "at-risk orders must not 5xx: %s", string(body))
	requireStatus(t, 200, status, body)

	data, ok := parseJSON(body)["data"].([]any)
	require.True(t, ok, "at-risk orders must serialize as a list: %s", string(body))

	validReasons := []string{"past_due", "undated", "short"}
	previousWeek := -1
	for _, raw := range data {
		row, ok := raw.(map[string]any)
		require.True(t, ok)

		assert.Equal(t, "schedule_order_coverage", jsonField(row, "object"))
		assert.Contains(t, validReasons, jsonField(row, "reason"), "every at-risk order must say why")

		order, ok := row["sales_order"].(map[string]any)
		require.True(t, ok, "an at-risk order must name the order: %v", row)
		assert.NotEmpty(t, jsonField(order, "id"))

		// The quantity at risk must be the shortfall, not a placeholder.
		units, ok := row["units_at_risk"].(float64)
		require.True(t, ok)
		assert.Positive(t, units, "an order reported at risk must have a quantity at risk")

		// Soonest first: that is the order they have to be worked.
		week, ok := row["due_week"].(float64)
		require.True(t, ok)
		assert.GreaterOrEqual(t, int(week), previousWeek, "at-risk orders must be sorted by due week")
		previousWeek = int(week)

		covering, ok := row["covering_lines"].(map[string]any)
		require.True(t, ok, "covering_lines must be a list resource, not null: %v", row["covering_lines"])
		lines, ok := covering["data"].([]any)
		require.True(t, ok)
		for _, rawLine := range lines {
			line, ok := rawLine.(map[string]any)
			require.True(t, ok)
			assert.Equal(t, "schedule_order_coverage_line", jsonField(line, "object"))
			allocated, ok := line["allocated_quantity"].(float64)
			require.True(t, ok)
			assert.Positive(t, allocated, "a campaign earmarked for an order must be earmarked for some quantity")
		}
	}
}

// A version that made no commitments cannot break any, so an empty answer is correct
// rather than an error.
func TestSchedulePromises_AtRiskOrdersRejectsUnknownSchedule(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(productionSchedulesPath+"/pnsc_doesnotexist00/at-risk-orders", nil)
	require.NoError(t, err)
	require.Less(t, status, 500, "an unknown schedule must not 5xx: %s", string(body))
	assert.Equal(t, 404, status, "body: %s", string(body))
}

// Capable-to-promise: a date drawn from the published plan, allowing for finishing.
//
// Publishes its own version rather than hoping one exists, so the quote path is
// actually exercised — a test that takes the "nothing published" branch proves only
// that the refusal works. Both quotes run against that one version: generating and
// publishing is a full solve, and doing it twice would hold the publishing lock
// twice as long for no extra coverage.
func TestSchedulePromises_QuoteComesFromThePublishedPlan(t *testing.T) {
	lockPublishing(t)

	schedule := ownedScheduleLocked(t, uniqueName("e2e-promise"))
	scheduleID := jsonField(schedule, "id")

	pubStatus, pubBody, err := apiClient.Put(schedulePath(scheduleID)+"/actions/publish", map[string]any{})
	require.NoError(t, err)
	requireStatus(t, 200, pubStatus, pubBody)
	require.Equal(t, "published", jsonField(parseJSON(pubBody), "status"))

	t.Run("quotes against the published version", func(t *testing.T) {
		status, body, err := apiClient.Post(quotePromiseDatePath, map[string]any{
			"item_id":  SeedGreigeItemID,
			"quantity": 1,
		}, newIdempotencyKey())
		require.NoError(t, err)
		require.Less(t, status, 500, "quoting must not 5xx: %s", string(body))
		requireStatus(t, 200, status, body)

		quote := parseJSON(body)
		assert.Equal(t, "promise_date_quote", jsonField(quote, "object"))
		assert.Equal(t, "1", jsonField(quote, "quantity"))

		quotedFrom, ok := quote["production_schedule"].(map[string]any)
		require.True(t, ok, "a quote must name the version it came from: %v", quote)
		assert.Equal(t, scheduleID, jsonField(quotedFrom, "id"),
			"the quote must come from the published version, not a draft")

		promisable, ok := quote["is_promisable"].(bool)
		require.True(t, ok)
		if promisable {
			assert.NotEmpty(t, jsonField(quote, "earliest_ship_date"),
				"a promisable quantity must carry a date")
		} else {
			assert.Empty(t, jsonField(quote, "earliest_ship_date"),
				"a quantity the horizon cannot supply must not be given a date")
		}
	})

	// A quantity far beyond what any plan holds is not promisable, and must come back
	// without a date rather than with an invented one.
	t.Run("an unreachable quantity gets no date", func(t *testing.T) {
		status, body, err := apiClient.Post(quotePromiseDatePath, map[string]any{
			"item_id":  SeedGreigeItemID,
			"quantity": 100000000,
		}, newIdempotencyKey())
		require.NoError(t, err)
		require.Less(t, status, 500, "must not 5xx: %s", string(body))
		requireStatus(t, 200, status, body)

		quote := parseJSON(body)
		assert.Equal(t, false, quote["is_promisable"],
			"a hundred million units is beyond any horizon")
		assert.Empty(t, jsonField(quote, "earliest_ship_date"))
	})
}

func TestSchedulePromises_QuoteValidatesItsInput(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		body map[string]any
	}{
		{"zero quantity", map[string]any{"item_id": SeedGreigeItemID, "quantity": 0}},
		{"negative quantity", map[string]any{"item_id": SeedGreigeItemID, "quantity": -5}},
		{"missing item", map[string]any{"quantity": 10}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			status, body, err := apiClient.Post(quotePromiseDatePath, tc.body, newIdempotencyKey())
			require.NoError(t, err)
			require.Less(t, status, 500, "must not 5xx: %s", string(body))
			assert.Equal(t, 400, status, "body: %s", string(body))
		})
	}
}
